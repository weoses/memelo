package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/storage"
)

const CheckDuplicateByHashKey = "CheckDuplicateByHashPipelineStep"

type CheckDuplicateByHashPipelineStep struct {
	BasePipelineStep

	metadataService storage.MetadataStorageService
}

func (s *CheckDuplicateByHashPipelineStep) Do(
	ctx context.Context,
	inputContext MetadataInputContext,
	pCtx *MetadataPipelineContext,
) error {
	var excludeIds []uuid.UUID
	if inputContext.SeedImageId != nil {
		excludeIds = append(excludeIds, *inputContext.SeedImageId)
	}

	items, _, err := s.metadataService.GetDuplicatesByHash(ctx, inputContext.AccountId, pCtx.Hash, excludeIds, nil, 1)
	if err != nil {
		return fmt.Errorf("error getting items by hash: %w", err)
	}
	if len(items) > 0 {
		pCtx.Duplicate = items[0]
	}
	return nil
}

func NewImageCheckDuplicateByHashPipelineStep(metadata storage.MetadataStorageService) ExtractPipelineStep {
	return &CheckDuplicateByHashPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 10,
			typ: []entity.MetadataType{entity.ImageMetadataType, entity.VideoMetadataType},
			key: CheckDuplicateByHashKey,
		},
		metadataService: metadata,
	}
}
