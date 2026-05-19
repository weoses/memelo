package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
	"github.com/weoses/memelo/storage-service/storage"
)

type CheckDuplicateByEmbeddingImgPipelineStep struct {
	BasePipelineStep

	metadata     storage.MetadataStorageService
	imageStorage storage.MediaStorageService
	extractor    ocr.LlmMediaExtractor
	searchConfig *conf.SearchConfig
	slogger      *slog.Logger
}

func (s *CheckDuplicateByEmbeddingImgPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	var excludeIds []uuid.UUID
	if inputContext.SeedImageId != nil {
		excludeIds = append(excludeIds, *inputContext.SeedImageId)
	}

	if len(pCtx.Embedding) == 0 {
		return fmt.Errorf("embedding is required")
	}

	items, _, err := s.metadata.GetDuplicatesByEmbeddingOrderByImageId(
		ctx,
		inputContext.AccountId,
		pCtx.Embedding[0],
		excludeIds,
		s.searchConfig.SemanticDuplicateThreshold,
		nil,
		1)
	if err != nil {
		return fmt.Errorf("error getting items by embedding: %w", err)
	}
	if len(items) == 0 {
		return nil
	}
	dupCandidate := items[0]

	dupImg, err := s.imageStorage.GetS3BackedData(ctx, dupCandidate.S3Id, storageMediaType(dupCandidate.Type, SavedOriginal))
	if err != nil {
		return fmt.Errorf("error wrapping duplicate candidate image: %w", err)
	}
	defer helper.QuietClose(dupImg, s.slogger)

	isDuplicate, err := s.extractor.CheckDuplicate(ctx, pCtx.ImageOriginalJpeg, dupImg)
	if err != nil {
		return fmt.Errorf("error checking duplicate via llm: %w", err)
	}
	if isDuplicate {
		pCtx.Duplicate = dupCandidate
	}

	return nil
}

func NewCheckDuplicateByEmbeddingImgPipelineStep(
	metadata storage.MetadataStorageService,
	imageStorage storage.MediaStorageService,
	extractor ocr.LlmMediaExtractor,
	cfg *conf.Config,
) ExtractPipelineStep {
	return &CheckDuplicateByEmbeddingImgPipelineStep{
		BasePipelineStep: BasePipelineStep{
			typ: []entity.MetadataType{entity.ImageMetadataType},
			pos: 40,
			key: CheckDuplicateByEmbeddingKey,
		},
		metadata:     metadata,
		imageStorage: imageStorage,
		extractor:    extractor,
		searchConfig: cfg.Search,
		slogger:      slog.With("step", "CheckDuplicateByEmbeddingImg"),
	}
}
