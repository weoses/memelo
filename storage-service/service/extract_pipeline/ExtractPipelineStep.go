package extract_pipeline

import (
	"context"

	"github.com/weoses/memelo/storage-service/service"
)

type ExtractPipelineStep interface {
	GetPos() int
	Check(ctx context.Context, steps *service.StepsToDo) (bool, error)
	Do(ctx context.Context, pipelineContext *service.PipelineContext) error
}
