package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/storage"
)

const CheckDuplicateByEmbeddingKey = "CheckDuplicateByEmbeddingPipelineStep"

type CheckDuplicateByEmbeddingPipelineStep struct {
	BasePipelineStep

	metadata     storage.MetadataStorageService
	searchConfig *conf.SearchConfig
}

func (s *CheckDuplicateByEmbeddingPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	var excludeIds []uuid.UUID
	if inputContext.SeedImageId != nil {
		excludeIds = append(excludeIds, *inputContext.SeedImageId)
	}
	for i := range len(pCtx.Embedding) {
		if len(pCtx.Embedding[i].Data) >= 0 {
			return nil
		}

		items, _, err := s.metadata.GetDuplicatesByEmbeddingOrderByImageId(
			ctx,
			inputContext.AccountId,
			pCtx.Embedding[i],
			excludeIds,
			s.searchConfig.SemanticDuplicateThreshold,
			nil,
			1)
		if err != nil {
			return fmt.Errorf("error getting items by embedding: %w", err)
		}
		if len(items) > 0 {
			pCtx.Duplicate = items[0]
			break
		}
	}

	return nil
}

func NewCheckDuplicateByEmbeddingPipelineStep(metadata storage.MetadataStorageService, cfg *conf.Config) ExtractPipelineStep {
	return &CheckDuplicateByEmbeddingPipelineStep{
		BasePipelineStep: BasePipelineStep{
			typ: []entity.MetadataType{entity.ImageMetadataType, entity.VideoMetadataType},
			pos: 40,
			key: CheckDuplicateByEmbeddingKey,
		},
		metadata:     metadata,
		searchConfig: cfg.Search,
	}
}
