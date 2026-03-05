package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/storage-service/storage"
)

type CheckDuplicateByHashPipelineStep struct {
	metadata storage.MetadataStorageService
}

func (s *CheckDuplicateByHashPipelineStep) GetPos() int {
	return 10
}

func (s *CheckDuplicateByHashPipelineStep) Check(_ context.Context, steps *StepsToDo) (bool, error) {
	return steps.DuplicateSearch && steps.CalcHash, nil
}

func (s *CheckDuplicateByHashPipelineStep) Do(ctx context.Context, pCtx *ImageMetadataPipelineContext) error {
	if pCtx.ImageHash == nil {
		return fmt.Errorf("error: image hash can't be nil")
	}

	items, err := s.metadata.GetByHash(ctx, pCtx.AccountId, *pCtx.ImageHash, helper.Addr(1))
	if err != nil {
		return fmt.Errorf("error getting items by hash: %w", err)
	}
	if len(items) > 0 {
		pCtx.Duplicate = items[0]
	}
	return nil
}

func NewCheckDuplicateByHashPipelineStep(metadata storage.MetadataStorageService) ExtractPipelineStep {
	return &CheckDuplicateByHashPipelineStep{metadata: metadata}
}
