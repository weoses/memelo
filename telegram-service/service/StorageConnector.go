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
	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"github.com/weoses/memelo/telegram-service/conf"
	"github.com/weoses/memelo/telegram-service/entity"
)

type StorageConnector interface {
	ProcessSearchQuery(
		ctx context.Context,
		accountId uuid.UUID,
		query string,
		pageSize int,
		pageAfterId *string,
	) ([]*entity.MemeSearchResult, error)

	CreateMeme(ctx context.Context, file []byte, mime string, accountId uuid.UUID) (*entity.MemeCreateResult, error)
	CreateVideo(ctx context.Context, file []byte, accountId uuid.UUID) (*entity.MemeCreateResult, error)

	DeleteMeme(ctx context.Context, accountId uuid.UUID, memeId uuid.UUID) error

	AddTag(ctx context.Context, accountId uuid.UUID, name string, description string) error
}

type StorageConnectorImpl struct {
	cl     v1connect.SearchServiceClient
	tagsCl v1connect.TagsServiceClient
	log    *slog.Logger
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
	query string,
	pageSize int,
	pageAfterId *string,
) ([]*entity.MemeSearchResult, error) {
	pageSize32 := int32(pageSize)
	response, err := s.cl.SearchMeme(ctx, &v1.SearchMemeRequest{
		AccountId:   accountId.String(),
		Query:       query,
		PageAfterId: pageAfterId,
		PageSize:    &pageSize32,
	})
	if err != nil {
		return nil, fmt.Errorf("storageService: search_pipeline meme query failed query: %s : %w", query, err)
	}

	entityResult := helper.TransformSlice(
		response.Results,
		make([]*entity.MemeSearchResult, len(response.Results)),
		func(dto *v1.MemeDto) *entity.MemeSearchResult {
			result := &entity.MemeSearchResult{
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
			return result
		})
	return entityResult, nil
}

func (s *StorageConnectorImpl) CreateMeme(ctx context.Context, file []byte, mime string, accountId uuid.UUID) (*entity.MemeCreateResult, error) {
	response, err := s.cl.CreateMeme(ctx, &v1.CreateMemeRequest{
		AccountId: accountId.String(),
		Image:     &v1.MediaDataDto{Data: file},
	})
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
	}, nil
}

func (s *StorageConnectorImpl) CreateVideo(ctx context.Context, file []byte, accountId uuid.UUID) (*entity.MemeCreateResult, error) {
	response, err := s.cl.CreateMeme(ctx, &v1.CreateMemeRequest{
		AccountId: accountId.String(),
		Video:     &v1.MediaDataDto{Data: file},
	})
	if err != nil {
		return nil, fmt.Errorf("storageService: create video meme failed: %w", err)
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
	}, nil
}

func NewStorageConnector(config *conf.Config) (StorageConnector, error) {
	cl := v1connect.NewSearchServiceClient(http.DefaultClient, config.StorageService.Uri)
	tagsCl := v1connect.NewTagsServiceClient(http.DefaultClient, config.StorageService.Uri)
	return &StorageConnectorImpl{
		cl:     cl,
		tagsCl: tagsCl,
		log:    slog.With("service", "StorageConnectorService"),
	}, nil
}
