package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

const VidCalcEmbeddingsKey = "VidCalcEmbeddingsPipelineStep"

type VidCalcEmbeddingsPipelineStep struct {
	BasePipelineStep

	embedder ocr.LlmEmbeddingExtractor
	slogger  *slog.Logger
}

func (s *VidCalcEmbeddingsPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	if len(pCtx.VideoSlices) == 0 {
		return nil
	}
	pCtx.Embedding = []entity.EmbeddingItem{}

	s.slogger.InfoContext(ctx, "calculating embeddings for video slices", "slices", len(pCtx.VideoSlices))

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}

	errorsList := make([]error, 0)
	embeddingsList := make([]entity.EmbeddingItem, 0)

	for i, slice := range pCtx.VideoSlices {
		errorsPtr := &errorsList
		embeddingsPtr := &embeddingsList

		wg.Go(func() {
			s.slogger.InfoContext(ctx, "embedding video slice", "index", i)
			sliceEmbeddings, err := s.embedder.GetVideoEmbedding(ctx, slice.Slice)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				*errorsPtr = append(*errorsPtr, fmt.Errorf("cannot get embedding for video slice %d: %w", i, err))
				return
			}
			*embeddingsPtr = append(*embeddingsPtr, sliceEmbeddings...)
		})
	}
	wg.Wait()

	if len(errorsList) > 0 {
		return errors.Join(errorsList...)
	}

	s.slogger.InfoContext(ctx, "embeddings done", "total", len(embeddingsList))
	pCtx.Embedding = embeddingsList
	return nil
}

func NewVidCalcEmbeddingsPipelineStep(embedder ocr.LlmEmbeddingExtractor) ExtractPipelineStep {
	return &VidCalcEmbeddingsPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 30,
			typ: []entity.MetadataType{entity.VideoMetadataType},
			key: VidCalcEmbeddingsKey,
		},
		embedder: embedder,
		slogger:  slog.With("service", "VidCalcEmbeddingsPipelineStep"),
	}
}
