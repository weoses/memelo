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
	CreateTag(ctx context.Context, tag string, description string) (*entity.ElasticTag, error)
	DeleteTag(ctx context.Context, id uuid.UUID) error
	ListTags(ctx context.Context, queryName *string, queryDescription *string) ([]entity.ElasticTag, error)
}

type TagServiceImpl struct {
	tagStorage                storage2.ElasticTagStorage
	tagMetadataExtractService TagMetadataExtractService
	slogger                   *slog.Logger
}

func (s *TagServiceImpl) CreateTag(ctx context.Context, tag string, description string) (*entity.ElasticTag, error) {
	existing, err := s.tagStorage.ListTag(ctx, &tag, nil)
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

func (s *TagServiceImpl) DeleteTag(ctx context.Context, id uuid.UUID) error {
	return s.tagStorage.DeleteTag(ctx, id)
}

func (s *TagServiceImpl) ListTags(ctx context.Context, queryName *string, queryDescription *string) ([]entity.ElasticTag, error) {
	return s.tagStorage.ListTag(ctx, queryName, queryDescription)
}

func NewTagService(storage storage2.ElasticTagStorage, tagMetadataExtractService TagMetadataExtractService) TagService {
	return &TagServiceImpl{
		tagStorage:                storage,
		tagMetadataExtractService: tagMetadataExtractService,
		slogger:                   slog.With("service", "TagService"),
	}
}
