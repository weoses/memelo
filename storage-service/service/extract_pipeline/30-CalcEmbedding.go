package extract_pipeline

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/ocr"
	"github.com/weoses/memelo/storage-service/service"
)

type CalcEmbeddingPipelineStep struct {
	imageEmbedder ocr.EmbeddingExtractor
}

func (s *CalcEmbeddingPipelineStep) GetPos() int {
	return 30
}

func (s *CalcEmbeddingPipelineStep) Check(_ context.Context, steps *service.StepsToDo) (bool, error) {
	return steps.CreateEmbedding, nil
}

func (s *CalcEmbeddingPipelineStep) Do(ctx context.Context, pCtx *service.PipelineContext) error {
	embedding, err := s.imageEmbedder.GetImageEmbeddingV1(ctx, pCtx.ImageRaw)
	if err != nil {
		return fmt.Errorf("error getting image embedding: %w", err)
	}
	pCtx.ImageEmbedding = embedding
	return nil
}

func NewCalcEmbeddingPipelineStep(imageEmbedder ocr.EmbeddingExtractor) service.PipelineStep {
	return &CalcEmbeddingPipelineStep{imageEmbedder: imageEmbedder}
}
