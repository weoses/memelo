package service

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"
)

type ImageMetadataExtractService interface {
	ProcessCreate(ctx context.Context, accountId uuid.UUID, raw []byte, checkDup bool) (*ImageMetadataPipelineContext, error)
}

type CreateImageServiceImpl struct {
	steps []ExtractPipelineStep
}

func (c *CreateImageServiceImpl) ProcessCreate(ctx context.Context, accountId uuid.UUID, raw []byte, checkDup bool) (*ImageMetadataPipelineContext, error) {
	pipelineCtx := &ImageMetadataPipelineContext{
		AccountId: accountId,
		ImageRaw:  raw,
	}

	for _, step := range c.steps {
		if err := step.Do(ctx, pipelineCtx); err != nil {
			return nil, fmt.Errorf("create pipeline: step failed (pos=%d): %w", step.GetPos(), err)
		}
		if checkDup && pipelineCtx.Duplicate != nil {
			return pipelineCtx, nil
		}
	}

	return pipelineCtx, nil
}

func NewImageMetadataExtractService(steps []ExtractPipelineStep) ImageMetadataExtractService {
	slices.SortFunc(steps, func(a, b ExtractPipelineStep) int {
		return a.GetPos() - b.GetPos()
	})
	return &CreateImageServiceImpl{steps: steps}
}
