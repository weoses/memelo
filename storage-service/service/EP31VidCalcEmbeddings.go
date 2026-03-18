package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

type VidCalcEmbeddingsPipelineStep struct {
	BasePipelineStep

	embedder ocr.EmbeddingExtractor
}

func (s *VidCalcEmbeddingsPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	if pCtx.VideoMp4 == nil {
		return nil
	}

	embeddings, err := s.embedder.GetVideoEmbedding(ctx, pCtx.VideoMp4)
	if err != nil {
		return fmt.Errorf("cannot get embedding for video: %w", err)
	}
	for _, e := range embeddings {
		pCtx.Embedding = append(pCtx.Embedding, *e)
	}
	return nil
}

func NewVidCalcEmbeddingsPipelineStep(embedder ocr.EmbeddingExtractor) ExtractPipelineStep {
	return &VidCalcEmbeddingsPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 31,
			typ: []entity.MetadataType{entity.VideoMetadataType},
		},
		embedder: embedder,
	}
}
