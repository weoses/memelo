package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

type ImageCalcEmbeddingPipelineStep struct {
	BasePipelineStep

	imageEmbedder ocr.LlmEmbeddingExtractor
}

func (s *ImageCalcEmbeddingPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	if !inputContext.ComputeEmbedding && len(pCtx.Embedding) > 0 {
		return nil
	}
	embedding, err := s.imageEmbedder.GetImageEmbedding(ctx, pCtx.ImageOriginalJpeg)
	if err != nil {
		return fmt.Errorf("error getting image embedding: %w", err)
	}
	pCtx.Embedding = append(pCtx.Embedding, *embedding)
	return nil
}

func NewImageCalcEmbeddingPipelineStep(imageEmbedder ocr.LlmEmbeddingExtractor) ExtractPipelineStep {
	return &ImageCalcEmbeddingPipelineStep{
		BasePipelineStep: BasePipelineStep{
			typ: []entity.MetadataType{entity.ImageMetadataType},
			pos: 30,
		},
		imageEmbedder: imageEmbedder}
}
