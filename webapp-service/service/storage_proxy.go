package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"github.com/weoses/memelo/webapp/conf"
)

type MemeResult struct {
	Id              string   `json:"id"`
	Caption         string   `json:"caption"`
	Type            string   `json:"type"`
	OcrResult       string   `json:"ocr_result"`
	OnScreenText    string   `json:"on_screen_text"`
	AudioTranscript string   `json:"audio_transcript"`
	AudioTrack      string   `json:"audio_track"`
	Tags            []string `json:"tags"`
	ThumbnailURL    string   `json:"thumbnail_url"`
	ThumbnailW      int32    `json:"thumbnail_w"`
	ThumbnailH      int32    `json:"thumbnail_h"`
	OriginalURL     string   `json:"original_url"`
	OriginalW       int32    `json:"original_w"`
	OriginalH       int32    `json:"original_h"`
	Status          string   `json:"status"`
	Edited          bool     `json:"edited"`
}

type UpdateParams struct {
	Caption         *string
	OnScreenText    *string
	AudioTranscript *string
	AudioTrack      *string
	OriginalS3Path  *string
	ThumbnailS3Path *string
}

type Pagination struct {
	Searcher     string   `json:"searcher"`
	SortingAfter []string `json:"sorting_after"`
}

type SearchResult struct {
	Memes    []MemeResult `json:"memes"`
	NextPage *Pagination  `json:"next_page"`
}

type StorageProxy interface {
	Search(ctx context.Context, accountId, query string, afterId *Pagination, limit int32) (*SearchResult, error)
	UploadByS3Path(ctx context.Context, accountId, s3path, mime string) (*MemeResult, error)
	GetMeme(ctx context.Context, accountId, id string) (*MemeResult, error)
	UpdateMeme(ctx context.Context, accountId, id string, params UpdateParams) (*MemeResult, error)
}

type storageProxy struct {
	cl  v1connect.SearchServiceClient
	log *slog.Logger
}

func NewStorageProxy(cfg *conf.Config) (StorageProxy, error) {
	cl := v1connect.NewSearchServiceClient(http.DefaultClient, cfg.StorageService.Uri)
	return &storageProxy{
		cl:  cl,
		log: slog.With("service", "storage_proxy"),
	}, nil
}

func (s *storageProxy) Search(ctx context.Context, accountId, query string, afterId *Pagination, limit int32) (*SearchResult, error) {
	req := &v1.SearchMemeRequest{
		AccountId: accountId,
		Query:     query,
		PageSize:  limit,
	}
	if afterId != nil && afterId.Searcher != "" {
		req.AfterId = &v1.PipelinePagination{
			Searcher:     afterId.Searcher,
			SortingAfter: afterId.SortingAfter,
		}
	}

	resp, err := s.cl.SearchMeme(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	memes := make([]MemeResult, len(resp.Results))
	for i, dto := range resp.Results {
		memes[i] = dtoToResult(dto)
	}

	result := &SearchResult{Memes: memes}
	if last := resp.GetLastId(); last != nil && last.GetSearcher() != "" {
		result.NextPage = &Pagination{
			Searcher:     last.GetSearcher(),
			SortingAfter: last.GetSortingAfter(),
		}
	}
	return result, nil
}

func (s *storageProxy) UploadByS3Path(ctx context.Context, accountId, s3path, mime string) (*MemeResult, error) {
	req := &v1.CreateMemeRequest{AccountId: accountId}
	if strings.HasPrefix(mime, "video/") {
		req.Video = &v1.MediaDataDto{S3Path: &s3path}
	} else {
		req.Image = &v1.MediaDataDto{S3Path: &s3path}
	}

	resp, err := s.cl.CreateMeme(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create meme by s3path failed: %w", err)
	}
	result := dtoToResult(resp.GetResult())
	switch resp.GetStatus() {
	case v1.CreateMemeStatus_STATUS_DUPLICATE:
		result.Status = "duplicate"
	default:
		result.Status = "new"
	}
	return &result, nil
}

func (s *storageProxy) GetMeme(ctx context.Context, accountId, id string) (*MemeResult, error) {
	resp, err := s.cl.GetMeme(ctx, &v1.GetMemeRequest{AccountId: accountId, Id: id})
	if err != nil {
		return nil, fmt.Errorf("get meme failed: %w", err)
	}
	result := dtoToResult(resp.GetResult())
	return &result, nil
}

func (s *storageProxy) UpdateMeme(ctx context.Context, accountId, id string, params UpdateParams) (*MemeResult, error) {
	req := &v1.UpdateMemeRequest{
		Id:        id,
		AccountId: accountId,
	}
	req.Caption = params.Caption
	req.OnScreenText = params.OnScreenText
	req.AudioTranscription = params.AudioTranscript
	req.AudioTrack = params.AudioTrack
	if params.OriginalS3Path != nil {
		req.Original = &v1.MediaDataDto{S3Path: params.OriginalS3Path}
	}
	if params.ThumbnailS3Path != nil {
		req.Thumbnail = &v1.MediaDataDto{S3Path: params.ThumbnailS3Path}
	}

	resp, err := s.cl.UpdateMeme(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("update meme failed: %w", err)
	}
	result := dtoToResult(resp.GetResult())
	return &result, nil
}

func dtoToResult(dto *v1.MemeDto) MemeResult {
	if dto == nil {
		return MemeResult{}
	}
	r := MemeResult{
		Id:              dto.GetId(),
		Caption:         dto.GetCaption(),
		Type:            dto.GetType(),
		OcrResult:       dto.GetOcrResult(),
		OnScreenText:    dto.GetOnScreenText(),
		AudioTranscript: dto.GetAudioTranscript(),
		AudioTrack:      dto.GetAudioTrack(),
		Tags:            dto.GetTags(),
	}
	if thumb := dto.GetImageThumbnail(); thumb != nil {
		r.ThumbnailURL = thumb.GetUrl()
		r.ThumbnailW = thumb.GetImageWidth()
		r.ThumbnailH = thumb.GetImageHeight()
	}
	if orig := dto.GetMediaOriginal(); orig != nil {
		r.OriginalURL = orig.GetUrl()
		r.OriginalW = orig.GetImageWidth()
		r.OriginalH = orig.GetImageHeight()
	}
	r.Edited = dto.GetEdited()
	return r
}
