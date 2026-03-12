package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/ocr"
)

type CalcEmbeddingPipelineStep struct {
	imageEmbedder ocr.EmbeddingExtractor
}

func (s *CalcEmbeddingPipelineStep) GetPos() int {
	return 30
}

func (s *CalcEmbeddingPipelineStep) Do(ctx context.Context, pCtx *ImageMetadataPipelineContext) error {
	embedding, err := s.imageEmbedder.GetImageEmbeddingV1(ctx, pCtx.ImageRaw)
	if err != nil {
		return fmt.Errorf("error getting image embedding: %w", err)
	}
	pCtx.ImageEmbedding = *embedding
	return nil
}

func NewCalcEmbeddingPipelineStep(imageEmbedder ocr.EmbeddingExtractor) ExtractPipelineStep {
	return &CalcEmbeddingPipelineStep{imageEmbedder: imageEmbedder}
}
