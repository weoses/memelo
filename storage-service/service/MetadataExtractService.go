package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"

	"github.com/weoses/memelo/storage-service/entity"
)

type MetadataExtractService interface {
	Extract(ctx context.Context, accountId uuid.UUID, typ entity.MetadataType, raw temp.S3BackedData, checkDup bool) (*MetadataPipelineContext, error)
}

type MetadataExtractServiceImpl struct {
	steps   []ExtractPipelineStep
	slogger *slog.Logger
}

func (c *MetadataExtractServiceImpl) Extract(ctx context.Context, accountId uuid.UUID, typ entity.MetadataType, raw temp.S3BackedData, checkDup bool) (*MetadataPipelineContext, error) {
	pipelineCtx := &MetadataPipelineContext{}
	inputCtx := MetadataInputContext{
		AccountId: accountId,
		Type:      typ,
		RawInput:  raw,
	}

	for _, step := range c.steps {
		if !slices.Contains(step.GetAllowedPipelineTypes(), typ) {
			continue
		}

		if err := step.Do(ctx, inputCtx, pipelineCtx); err != nil {
			helper.QuietClose(pipelineCtx, c.slogger)
			return nil, fmt.Errorf("create pipeline: step failed (pos=%d): %w", step.GetPos(), err)
		}
		if checkDup && pipelineCtx.Duplicate != nil {
			return pipelineCtx, nil
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
