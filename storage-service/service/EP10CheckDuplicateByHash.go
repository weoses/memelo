package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/storage"
)

type ImageCheckDuplicateByHashPipelineStep struct {
	BasePipelineStep

	metadataService storage.MetadataStorageService
}

func (s *ImageCheckDuplicateByHashPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	items, _, err := s.metadataService.GetByHash(ctx, inputContext.AccountId, pCtx.Hash, nil, 1)
	if err != nil {
		return fmt.Errorf("error getting items by hash: %w", err)
	}
	if len(items) > 0 {
		pCtx.Duplicate = items[0]
	}
	return nil
}

func NewImageCheckDuplicateByHashPipelineStep(metadata storage.MetadataStorageService) ExtractPipelineStep {
	return &ImageCheckDuplicateByHashPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 10,
			typ: []entity.MetadataType{entity.ImageMetadataType, entity.VideoMetadataType},
		},
		metadataService: metadata,
	}
}
