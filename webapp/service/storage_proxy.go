package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"github.com/weoses/memelo/webapp/conf"
)

type MemeResult struct {
	Id           string   `json:"id"`
	Caption      string   `json:"caption"`
	Type         string   `json:"type"`
	OcrResult    string   `json:"ocr_result"`
	Tags         []string `json:"tags"`
	ThumbnailURL string   `json:"thumbnail_url"`
	ThumbnailW   int32    `json:"thumbnail_w"`
	ThumbnailH   int32    `json:"thumbnail_h"`
	OriginalURL  string   `json:"original_url"`
	OriginalW    int32    `json:"original_w"`
	OriginalH    int32    `json:"original_h"`
	SortingId    int64    `json:"sorting_id"`
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
	Upload(ctx context.Context, accountId string, filename string, r io.Reader, mime string) (*MemeResult, error)
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

func (s *storageProxy) Upload(ctx context.Context, accountId string, filename string, r io.Reader, mime string) (*MemeResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading upload: %w", err)
	}

	req := &v1.CreateMemeRequest{AccountId: accountId}
	if strings.HasPrefix(mime, "video/") {
		req.Video = &v1.MediaDataDto{Data: data}
	} else {
		req.Image = &v1.MediaDataDto{Data: data}
	}

	resp, err := s.cl.CreateMeme(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create meme failed: %w", err)
	}

	result := dtoToResult(resp.GetResult())
	return &result, nil
}

func dtoToResult(dto *v1.MemeDto) MemeResult {
	if dto == nil {
		return MemeResult{}
	}
	r := MemeResult{
		Id:        dto.GetId(),
		Caption:   dto.GetCaption(),
		Type:      dto.GetType(),
		OcrResult: dto.GetOcrResult(),
		Tags:      dto.GetTags(),
		SortingId: dto.GetSortingId(),
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
	return r
}
