package extract_pipeline

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/service"
)

type CheckDuplicateByEmbeddingPipelineStep struct {
	metadata service.MetadataStorageService
}

func (s *CheckDuplicateByEmbeddingPipelineStep) GetPos() int {
	return 40
}

func (s *CheckDuplicateByEmbeddingPipelineStep) Check(_ context.Context, steps *service.StepsToDo) (bool, error) {
	return steps.DuplicateSearch && steps.CreateEmbedding, nil
}

func (s *CheckDuplicateByEmbeddingPipelineStep) Do(ctx context.Context, pCtx *service.PipelineContext) error {
	if pCtx.ImageEmbedding == nil {
		return fmt.Errorf("error: image embedding can't be nil")
	}

	items, err := s.metadata.SearchByEmbeddingV1(ctx, pCtx.AccountId, pCtx.ImageEmbedding, 1, true)
	if err != nil {
		return fmt.Errorf("error getting items by embedding: %w", err)
	}
	if len(items) > 0 {
		pCtx.Duplicate = items[0]
	}
	return nil
}

func NewCheckDuplicateByEmbeddingPipelineStep(metadata service.MetadataStorageService) service.PipelineStep {
	return &CheckDuplicateByEmbeddingPipelineStep{metadata: metadata}
}
