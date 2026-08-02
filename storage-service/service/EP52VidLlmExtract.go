package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

const VidLlmExtractKey = "VidLlmExtractPipelineStep"

type VidLlmExtractPipelineStep struct {
	BasePipelineStep

	extractor ocr.LlmMediaExtractor
	slogger   *slog.Logger
}

func (s *VidLlmExtractPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	type OneSliceResult struct {
		SliceNumber    int
		SliceStartTime int
		SliceEndTime   int
		Result         ocr.MediaExtractResult
	}

	if len(pCtx.VideoSlices) == 0 {
		return errors.New("no video slices")
	}
	s.slogger.InfoContext(ctx, "processing video slices", "slices", len(pCtx.VideoSlices))

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	errs := make([]error, 0)
	results := make([]OneSliceResult, 0)

	for _, slice := range pCtx.VideoSlices {
		wg.Go(func() {
			s.processSlice(ctx,
				slice,
				func(err error) {
					mu.Lock()
					defer mu.Unlock()
					errs = append(errs, err)
				},
				func(result ocr.MediaExtractResult) {
					mu.Lock()
					defer mu.Unlock()
					results = append(results, OneSliceResult{
						SliceNumber:    slice.SliceNumber,
						SliceStartTime: slice.SliceStartTime,
						SliceEndTime:   slice.SliceEndTime,
						Result:         result,
					})
				})
		})
	}
	wg.Wait()

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	slices.SortFunc(results, func(e OneSliceResult, e2 OneSliceResult) int {
		return e.SliceNumber - e2.SliceNumber
	})

	var combined *ocr.MediaExtractResult
	if len(results) == 1 {
		s.slogger.InfoContext(ctx, "extraction done")
		s.slogger.InfoContext(ctx, "single slice, skipping combine")
		combined = &results[0].Result
	} else {
		var err error
		s.slogger.InfoContext(ctx, "combining slice results", "count", len(results))
		ocrResultPerSlices := helper.TransformSlice(
			results,
			make([]ocr.MediaExtractResult, len(results)),
			func(result OneSliceResult) ocr.MediaExtractResult {
				return result.Result
			})
		combined, err = s.extractor.CombineResults(ctx, ocrResultPerSlices)
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

	pCtx.ResultPerVideoSlices = helper.TransformSlice(
		results,
		make([]entity.ResultPerVideoSlice, len(results)),
		func(result OneSliceResult) entity.ResultPerVideoSlice {
			return entity.ResultPerVideoSlice{
				SliceNumber:    result.SliceNumber,
				SliceStartTime: result.SliceStartTime,
				SliceEndTime:   result.SliceEndTime,
				Result: entity.Result{
					OnScreenText:    result.Result.OnScreenText,
					AudioTranscript: result.Result.AudioTranscript,
					Caption:         result.Result.Caption,
					AudioTrack:      result.Result.AudioTrack,
				},
			}
		})

	return nil
}

func (s *VidLlmExtractPipelineStep) processSlice(
	ctx context.Context,
	slice VideoSlice,
	appendErr func(err error),
	appendResult func(ocr.MediaExtractResult)) {
	s.slogger.InfoContext(ctx, "processing video slice part - extract llm data",
		"index", slice.SliceNumber,
		"startTime", slice.SliceStartTime,
		"endTime", slice.SliceEndTime)

	r, err := s.extractor.ProcessVideo(ctx, slice.Slice)
	if err != nil {
		appendErr(err)
		return
	}
	s.slogger.DebugContext(ctx, "processed slice",
		"index", slice.SliceNumber,
		"onScreenText", r.OnScreenText,
		"audioTransription", r.AudioTranscript)

	appendResult(ocr.MediaExtractResult{
		OnScreenText:    r.OnScreenText,
		Caption:         r.Caption,
		AudioTranscript: r.AudioTranscript,
		AudioTrack:      r.AudioTrack,
	})
}

func NewVidLlmExtractPipelineStep(extractor ocr.LlmMediaExtractor) ExtractPipelineStep {
	return &VidLlmExtractPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 52,
			typ: []entity.MetadataType{entity.VideoMetadataType},
			key: VidLlmExtractKey,
		},
		extractor: extractor,
		slogger:   slog.With("service", "VidLlmExtractPipelineStep"),
	}
}
