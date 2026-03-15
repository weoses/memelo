package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
	storage2 "github.com/weoses/memelo/storage-service/storage"
)

var ErrTagDuplicate = errors.New("tag already exists")

type TagService interface {
	CreateTag(ctx context.Context, accountId uuid.UUID, tag string, description string) (*entity.ElasticTag, error)
	DeleteTag(ctx context.Context, accountId uuid.UUID, id uuid.UUID) error
	DeleteAll(ctx context.Context, accountId uuid.UUID) error
	ListTags(ctx context.Context, accountId uuid.UUID, queryName *string, queryDescription *string) ([]entity.ElasticTag, error)
}

type TagServiceImpl struct {
	tagStorage                storage2.ElasticTagStorage
	tagMetadataExtractService TagMetadataExtractService
	slogger                   *slog.Logger
}

func (s *TagServiceImpl) CreateTag(ctx context.Context, accountId uuid.UUID, tag string, description string) (*entity.ElasticTag, error) {
	existing, err := s.tagStorage.ListTag(ctx, accountId, &tag, nil)
	if err != nil {
		return nil, fmt.Errorf("list tags failed: %w", err)
	}
	for _, e := range existing {
		if e.Tag == tag {
			return nil, ErrTagDuplicate
		}
	}

	metadata, err := s.tagMetadataExtractService.ProcessTagMetadata(ctx, tag, description)
	if err != nil {
		return nil, fmt.Errorf("process tag metadata failed: %w", err)
	}

	tagEntity := entity.ElasticTag{
		Id:          uuid.New(),
		AccountId:   accountId,
		Tag:         tag,
		Description: description,
		EmbeddingV1: metadata.Embedding,
		Created:     time.Now().UnixMicro(),
	}

	if err := s.tagStorage.SaveTag(ctx, tagEntity); err != nil {
		return nil, fmt.Errorf("save tag failed: %w", err)
	}

	return &tagEntity, nil
}

func (s *TagServiceImpl) DeleteTag(ctx context.Context, accountId uuid.UUID, id uuid.UUID) error {
	return s.tagStorage.DeleteTag(ctx, accountId, id)
}

func (s *TagServiceImpl) DeleteAll(ctx context.Context, accountId uuid.UUID) error {
	return s.tagStorage.DeleteAllTags(ctx, accountId)
}

func (s *TagServiceImpl) ListTags(ctx context.Context, accountId uuid.UUID, queryName *string, queryDescription *string) ([]entity.ElasticTag, error) {
	return s.tagStorage.ListTag(ctx, accountId, queryName, queryDescription)
}

func NewTagService(storage storage2.ElasticTagStorage, tagMetadataExtractService TagMetadataExtractService) TagService {
	return &TagServiceImpl{
		tagStorage:                storage,
		tagMetadataExtractService: tagMetadataExtractService,
		slogger:                   slog.With("service", "TagService"),
	}
}
