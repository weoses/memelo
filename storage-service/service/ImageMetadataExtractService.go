package service

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/service/extract_pipeline"
)

type PipelineContext struct {
	AccountId          uuid.UUID
	ImageHash          *string
	ImageEmbedding     *entity.ElasticEmbeddingV1
	ImageOcrResult     *string
	ImageThumbnail     *[]byte
	ImageThumbnailSize *entity.ElasticSizes
	ImageRaw           []byte
	ImageRawSize       *entity.ElasticSizes
	Duplicate          *entity.ElasticImageMetaData
}

type StepsToDo struct {
	DuplicateSearch bool
	Ocr             bool
	CreateThumbnail bool
	CalcSize        bool
	CreateEmbedding bool
	CalcHash        bool
}

type ImageMetadataExtractService interface {
	ProcessCreate(ctx context.Context, accountId uuid.UUID, steps *StepsToDo, raw []byte) (*PipelineContext, error)
}

type CreateImageServiceImpl struct {
	steps []extract_pipeline.ExtractPipelineStep
}

func (c *CreateImageServiceImpl) ProcessCreate(ctx context.Context, accountId uuid.UUID, steps *StepsToDo, raw []byte) (*PipelineContext, error) {
	pipelineCtx := &PipelineContext{
		AccountId: accountId,
		ImageRaw:  raw,
	}

	for _, step := range c.steps {
		shouldRun, err := step.Check(ctx, steps)
		if err != nil {
			return nil, fmt.Errorf("create pipeline: step check failed (pos=%d): %w", step.GetPos(), err)
		}
		if !shouldRun {
			continue
		}
		if err := step.Do(ctx, pipelineCtx); err != nil {
			return nil, fmt.Errorf("create pipeline: step failed (pos=%d): %w", step.GetPos(), err)
		}
		if pipelineCtx.Duplicate != nil {
			return pipelineCtx, nil
		}
	}

	return pipelineCtx, nil
}

func NewImageMetadataExtractService(steps []extract_pipeline.ExtractPipelineStep) ImageMetadataExtractService {
	slices.SortFunc(steps, func(a, b extract_pipeline.ExtractPipelineStep) int {
		return a.GetPos() - b.GetPos()
	})
	return &CreateImageServiceImpl{steps: steps}
}
