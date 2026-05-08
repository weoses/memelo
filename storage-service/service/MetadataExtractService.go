package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/weoses/memelo/common/helper"
)

type MetadataExtractService interface {
	Extract(ctx context.Context, inputCtx MetadataInputContext) (*MetadataPipelineContext, error)
}

type MetadataExtractServiceImpl struct {
	steps   []ExtractPipelineStep
	slogger *slog.Logger
}

func (c *MetadataExtractServiceImpl) Extract(ctx context.Context, inputCtx MetadataInputContext) (*MetadataPipelineContext, error) {
	var pipelineCtx = &MetadataPipelineContext{
		Embedding: inputCtx.SeedEmbeddings,
	}

	for _, step := range c.steps {
		if !slices.Contains(step.GetAllowedPipelineTypes(), inputCtx.Type) {
			continue
		}

		if !inputCtx.StepCallback(ctx, step.GetKey(), pipelineCtx) {
			return pipelineCtx, nil
		}

		if err := step.Do(ctx, inputCtx, pipelineCtx); err != nil {
			helper.QuietClose(pipelineCtx, c.slogger)
			return nil, fmt.Errorf("create pipeline: step failed (pos=%d): %w", step.GetPos(), err)
		}
	}

	return pipelineCtx, nil
}

func NewImageMetadataExtractService(steps []ExtractPipelineStep) MetadataExtractService {
	slices.SortFunc(steps, func(a, b ExtractPipelineStep) int {
		return a.GetPos() - b.GetPos()
	})
	return &MetadataExtractServiceImpl{
		steps:   steps,
		slogger: slog.With("service", "MetadataExtractService"),
	}
}
