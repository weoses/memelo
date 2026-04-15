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

type VidLlmExtractPipelineStep struct {
	BasePipelineStep

	extractor ocr.LlmMediaExtractor
	slogger   *slog.Logger
}

func (s *VidLlmExtractPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	if len(pCtx.VideoSlices) == 0 {
		return errors.New("no video slices")
	}

	s.slogger.InfoContext(ctx, "processing video slices", "slices", len(pCtx.VideoSlices))

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	errs := make([]error, 0)
	results := make([]*ocr.MediaExtractResult, len(pCtx.VideoSlices))

	for i, slice := range pCtx.VideoSlices {
		wg.Go(func() {
			s.slogger.InfoContext(ctx, "processing slice", "index", i)
			r, err := s.extractor.ProcessVideo(ctx, slice)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("slice %d: %w", i, err))
				return
			}
			results[i] = r
		})
	}
	wg.Wait()

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	var combined *ocr.MediaExtractResult
	if len(results) == 1 {
		s.slogger.InfoContext(ctx, "single slice, skipping combine")
		combined = results[0]
	} else {
		s.slogger.InfoContext(ctx, "combining slice results", "count", len(results))
		var err error
		combined, err = s.extractor.CombineResults(ctx, results)
		if err != nil {
			return fmt.Errorf("combine results: %w", err)
		}
	}

	s.slogger.InfoContext(ctx, "extraction done")
	pCtx.Result = &entity.Result{
		OnScreenText:    combined.OnScreenText,
		AudioTranscript: combined.AudioTranscript,
		AudioTrack:      combined.AudioTrack,
		Caption:         combined.Caption,
	}
	return nil
}

func NewVidLlmExtractPipelineStep(extractor ocr.LlmMediaExtractor) ExtractPipelineStep {
	return &VidLlmExtractPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 52,
			typ: []entity.MetadataType{entity.VideoMetadataType},
		},
		extractor: extractor,
		slogger:   slog.With("service", "VidLlmExtractPipelineStep"),
	}
}
