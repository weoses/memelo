package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"
	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"github.com/weoses/memelo/telegram-service/conf"
	"github.com/weoses/memelo/telegram-service/entity"
)

type StorageConnector interface {
	ProcessSearchQuery(
		ctx context.Context,
		accountId uuid.UUID,
		params entity.SearchParams,
		pageSize int,
	) (*entity.SearchQueryResult, error)

	CreateMeme(ctx context.Context, data temp.S3BackedData, reqType string, accountId uuid.UUID) (*entity.MemeCreateResult, error)

	DeleteMeme(ctx context.Context, accountId uuid.UUID, memeId uuid.UUID) error

	AddTag(ctx context.Context, accountId uuid.UUID, name string, description string) error

	GetRandomMeme(ctx context.Context, accountId uuid.UUID, mediaType string) (*entity.MemeSearchResult, error)

	StartRecomputeById(ctx context.Context, accountId uuid.UUID, memeId uuid.UUID) error
}

type StorageConnectorImpl struct {
	cl          v1connect.SearchServiceClient
	tagsCl      v1connect.TagsServiceClient
	recomputeCl v1connect.RecomputeServiceClient
	log         *slog.Logger
}

func (s *StorageConnectorImpl) AddTag(ctx context.Context, accountId uuid.UUID, name string, description string) error {
	_, err := s.tagsCl.CreateTag(ctx, &v1.CreateTagRequest{
		AccountId:   accountId.String(),
		Tag:         name,
		Description: description,
	})
	if err != nil {
		var connectErr *connect.Error
		if errors.As(err, &connectErr) && connectErr.Code() == connect.CodeAlreadyExists {
			return fmt.Errorf("tag '%s' already exists", name)
		}
		return fmt.Errorf("AddTag failed: name=%s %w", name, err)
	}
	return nil
}

func (s *StorageConnectorImpl) DeleteMeme(ctx context.Context, accountId uuid.UUID, memeId uuid.UUID) error {
	_, err := s.cl.DeleteMeme(ctx, &v1.DeleteMemeRequest{
		AccountId: accountId.String(),
		Id:        memeId.String(),
	})
	if err != nil {
		return fmt.Errorf("DeleteMeme failed: accountId=%s memeId=%s %w", accountId, memeId, err)
	}
	return nil
}

func (s *StorageConnectorImpl) ProcessSearchQuery(
	ctx context.Context,
	accountId uuid.UUID,
	params entity.SearchParams,
	pageSize int,
) (*entity.SearchQueryResult, error) {
	req := &v1.SearchMemeRequest{
		AccountId: accountId.String(),
		Query:     params.Query,
		PageSize:  int32(pageSize),
		Type:      params.Type,
	}
	if params.Pagination != nil {
		req.AfterId = &v1.PipelinePagination{
			Searcher:     params.Pagination.Searcher,
			SortingAfter: params.Pagination.SortingAfter,
		}
	}

	response, err := s.cl.SearchMeme(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("storageService: search_pipeline meme query failed query: %s : %w", params.Query, err)
	}

	results := helper.TransformSlice(
		response.Results,
		make([]entity.MemeSearchResult, len(response.Results)),
		func(dto *v1.MemeDto) entity.MemeSearchResult {
			result := entity.MemeSearchResult{
				Id: dto.GetId(),
			}
			if dto.GetMediaOriginal() != nil {
				result.MediaUrl = dto.GetMediaOriginal().GetUrl()
				result.MediaWidth = int(dto.GetMediaOriginal().GetImageWidth())
				result.MediaHeight = int(dto.GetMediaOriginal().GetImageHeight())
			}
			if dto.GetImageThumbnail() != nil {
				result.ThumbUrl = dto.GetImageThumbnail().GetUrl()
				result.ThumbWidth = int(dto.GetImageThumbnail().GetImageWidth())
				result.ThumbHeight = int(dto.GetImageThumbnail().GetImageHeight())
			}
			result.Type = dto.GetType()
			result.Caption = dto.GetCaption()
			return result
		})

	qr := &entity.SearchQueryResult{Results: results}
	if last := response.GetLastId(); last != nil {
		qr.Pagination = &entity.PaginationOffset{
			Searcher:     last.GetSearcher(),
			SortingAfter: last.GetSortingAfter(),
		}
	}
	return qr, nil
}

func (s *StorageConnectorImpl) CreateMeme(ctx context.Context, data temp.S3BackedData, reqType string, accountId uuid.UUID) (*entity.MemeCreateResult, error) {
	s3path, err := data.GetS3Path(ctx)
	if err != nil {
		return nil, fmt.Errorf("storageService: get s3 path failed: %w", err)
	}

	req := &v1.CreateMemeRequest{AccountId: accountId.String()}
	switch reqType {
	case "video":
		req.Video = &v1.MediaDataDto{S3Path: &s3path}
	case "image":
		req.Image = &v1.MediaDataDto{S3Path: &s3path}
	default:
		return nil, fmt.Errorf("storageService: unknown type: %s", reqType)
	}

	response, err := s.cl.CreateMeme(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("storageService: create meme failed: %w", err)
	}

	memeId, err := uuid.Parse(response.Result.GetId())
	if err != nil {
		return nil, fmt.Errorf("storageService: failed to parse created meme id: %s: %w", response.Result.GetId(), err)
	}
	return &entity.MemeCreateResult{
		Id:              memeId,
		Text:            response.Result.GetOcrResult(),
		DuplicateStatus: response.Status.String(),
		Tags:            response.Result.GetTags(),
		Caption:         response.Result.GetCaption(),
	}, nil
}

func (s *StorageConnectorImpl) GetRandomMeme(ctx context.Context, accountId uuid.UUID, mediaType string) (*entity.MemeSearchResult, error) {
	req := &v1.GetRandomMemeRequest{AccountId: accountId.String()}
	if mediaType != "" {
		req.Type = &mediaType
	}

	response, err := s.cl.GetRandomMeme(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("StorageConnector: GetRandomMeme failed: %w", err)
	}

	dto := response.GetResult()
	if dto == nil {
		return nil, nil
	}

	result := &entity.MemeSearchResult{
		Id:      dto.GetId(),
		Type:    dto.GetType(),
		Caption: dto.GetCaption(),
	}
	if dto.GetMediaOriginal() != nil {
		result.MediaUrl = dto.GetMediaOriginal().GetUrl()
		result.MediaWidth = int(dto.GetMediaOriginal().GetImageWidth())
		result.MediaHeight = int(dto.GetMediaOriginal().GetImageHeight())
	}
	if dto.GetImageThumbnail() != nil {
		result.ThumbUrl = dto.GetImageThumbnail().GetUrl()
		result.ThumbWidth = int(dto.GetImageThumbnail().GetImageWidth())
		result.ThumbHeight = int(dto.GetImageThumbnail().GetImageHeight())
	}
	return result, nil
}

func (s *StorageConnectorImpl) StartRecomputeById(ctx context.Context, accountId uuid.UUID, memeId uuid.UUID) error {
	_, err := s.recomputeCl.StartRecomputeJobById(ctx, &v1.RecomputeOneRequest{
		AccountId: accountId.String(),
		MediaId:   memeId.String(),
		Options: &v1.RecomputeOptions{
			ComputeExtractor:   true,
			ComputeEmbedding:   true,
			UpdateStorageItems: true,
		},
	})
	if err != nil {
		return fmt.Errorf("StartRecomputeById failed: accountId=%s memeId=%s: %w", accountId, memeId, err)
	}
	return nil
}

func NewStorageConnector(config *conf.Config) (StorageConnector, error) {
	cl := v1connect.NewSearchServiceClient(http.DefaultClient, config.StorageService.Uri)
	tagsCl := v1connect.NewTagsServiceClient(http.DefaultClient, config.StorageService.Uri)
	recomputeCl := v1connect.NewRecomputeServiceClient(http.DefaultClient, config.StorageService.Uri)
	return &StorageConnectorImpl{
		cl:          cl,
		tagsCl:      tagsCl,
		recomputeCl: recomputeCl,
		log:         slog.With("service", "StorageConnectorService"),
	}, nil
}
