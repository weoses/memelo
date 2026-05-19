package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/storage"
)

const CheckDuplicateByEmbeddingKey = "CheckDuplicateByEmbeddingPipelineStep"

type CheckDuplicateByEmbeddingVideoPipelineStep struct {
	BasePipelineStep

	metadata     storage.MetadataStorageService
	searchConfig *conf.SearchConfig
	slogger      *slog.Logger
}

func (s *CheckDuplicateByEmbeddingVideoPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	var excludeIds []uuid.UUID
	if inputContext.SeedImageId != nil {
		excludeIds = append(excludeIds, *inputContext.SeedImageId)
	}

	lastMatched := make(map[uuid.UUID]int)

	for i := range len(pCtx.Embedding) {
		if len(pCtx.Embedding[i].Data) == 0 {
			return nil
		}
		s.slogger.InfoContext(ctx, "Checking slice embedding", "i", i)

		items, _, err := s.metadata.GetDuplicatesByEmbeddingOrderByImageId(
			ctx,
			inputContext.AccountId,
			pCtx.Embedding[i],
			excludeIds,
			s.searchConfig.SemanticDuplicateThreshold,
			nil,
			30)
		if err != nil {
			return fmt.Errorf("error getting items by embedding: %w", err)
		}

		for _, item := range items {
			lastMatched[item.ImageId]++
		}
	}

	allEmbeddingsLen := float64(len(pCtx.Embedding))
	for id, idMatchedCount := range lastMatched {
		s.slogger.InfoContext(ctx, "Check matched duplicate", "id", id, "idMatchedCount", idMatchedCount)
		if float64(idMatchedCount)/allEmbeddingsLen >= s.searchConfig.PercentageDuplicatePartsThreshold {
			dupById, err := s.metadata.GetById(ctx, inputContext.AccountId, id)
			if err != nil {
				return fmt.Errorf("error getting duplicate record by id: %w", err)
			}
			pCtx.Duplicate = dupById
			break
		}
	}

	return nil
}

func NewCheckDuplicateByEmbeddingVidPipelineStep(metadata storage.MetadataStorageService, cfg *conf.Config) ExtractPipelineStep {
	return &CheckDuplicateByEmbeddingVideoPipelineStep{
		BasePipelineStep: BasePipelineStep{
			typ: []entity.MetadataType{entity.VideoMetadataType},
			pos: 40,
			key: CheckDuplicateByEmbeddingKey,
		},
		metadata:     metadata,
		searchConfig: cfg.Search,
		slogger:      slog.With("service", "CheckDuplicateByEmbeddingVideoPipelineStep"),
	}
}
