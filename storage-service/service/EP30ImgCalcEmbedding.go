package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

const ImgCalcEmbeddingKey = "ImgCalcEmbeddingPipelineStep"

type ImageCalcEmbeddingPipelineStep struct {
	BasePipelineStep

	imageEmbedder ocr.LlmEmbeddingExtractor
}

func (s *ImageCalcEmbeddingPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
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
			key: ImgCalcEmbeddingKey,
		},
		imageEmbedder: imageEmbedder}
}
