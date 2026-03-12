package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/storage"
)

type CheckDuplicateByEmbeddingPipelineStep struct {
	metadata storage.MetadataStorageService
}

func (s *CheckDuplicateByEmbeddingPipelineStep) GetPos() int {
	return 40
}

func (s *CheckDuplicateByEmbeddingPipelineStep) Do(ctx context.Context, pCtx *ImageMetadataPipelineContext) error {

	items, err := s.metadata.SearchByEmbeddingV1(ctx, pCtx.AccountId, pCtx.ImageEmbedding, 1, true)
	if err != nil {
		return fmt.Errorf("error getting items by embedding: %w", err)
	}
	if len(items) > 0 {
		pCtx.Duplicate = items[0]
	}
	return nil
}

func NewCheckDuplicateByEmbeddingPipelineStep(metadata storage.MetadataStorageService) ExtractPipelineStep {
	return &CheckDuplicateByEmbeddingPipelineStep{metadata: metadata}
}
