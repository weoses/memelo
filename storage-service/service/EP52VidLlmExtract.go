package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

const VidLlmExtractKey = "VidLlmExtractPipelineStep"

type VidLlmExtractPipelineStep struct {
	BasePipelineStep

	extractor     ocr.LlmMediaExtractor
	video2audio   ocr.Video2AudioConverter
	separateAudio bool
	slogger       *slog.Logger
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
	ctxInProcess, cancel := context.WithCancel(ctx)
	defer cancel()

	var videoResult *ocr.MediaExtractResult
	var audioResult *ocr.MediaExtractResult

	wg := sync.WaitGroup{}
	wg.Go(func() {
		s.slogger.InfoContext(ctxInProcess, "processing video slice part - extract llm data",
			"index", slice.SliceNumber,
			"startTime", slice.SliceStartTime,
			"endTime", slice.SliceEndTime)

		var videoData temp.Data = slice.Slice
		if s.separateAudio {
			cut, err := s.video2audio.CutAudio(ctxInProcess, slice.Slice)
			defer helper.QuietClose(cut, s.slogger)
			if err != nil {
				appendErr(err)
				cancel()
				return
			}
			videoData = cut
		}

		r, err := s.extractor.ProcessVideo(ctxInProcess, videoData)
		if err != nil {
			appendErr(err)
			cancel()
			return
		}
		videoResult = r
		s.slogger.DebugContext(ctxInProcess, "processed slice",
			"index", slice.SliceNumber,
			"onScreenText", r.OnScreenText,
			"audioTransription", r.AudioTranscript)
	})
	wg.Go(func() {
		if !s.separateAudio {
			return
		}

		s.slogger.InfoContext(ctxInProcess, "processing audio slice part - extract mp3",
			"index", slice.SliceNumber,
			"startTime", slice.SliceStartTime,
			"endTime", slice.SliceEndTime)

		audioData, err := s.video2audio.ExtractAudio(ctxInProcess, slice.Slice)
		defer helper.QuietClose(audioData, s.slogger)
		if err != nil {
			appendErr(err)
			cancel()
			return
		}

		s.slogger.InfoContext(ctxInProcess, "processing audio slice part - extract llm data",
			"index", slice.SliceNumber,
			"startTime", slice.SliceStartTime,
			"endTime", slice.SliceEndTime)

		r, err := s.extractor.ProcessAudio(ctxInProcess, audioData)
		if err != nil {
			appendErr(err)
			cancel()
		}
		audioResult = r
		s.slogger.DebugContext(ctxInProcess, "processed audio slice part",
			"index", slice.SliceNumber,
			"onScreenText", r.OnScreenText,
			"audioTransription", r.AudioTranscript)
	})

	wg.Wait()
	select {
	case <-ctxInProcess.Done():
		return
	default:
	}

	result := ocr.MediaExtractResult{}
	if videoResult != nil {
		result.OnScreenText = videoResult.OnScreenText
		result.Caption = videoResult.Caption
		result.AudioTranscript = videoResult.AudioTranscript
		result.AudioTrack = videoResult.AudioTrack
	}

	if audioResult != nil {
		result.AudioTranscript = audioResult.AudioTranscript
		result.AudioTrack = audioResult.AudioTrack
	}

	appendResult(result)
}

func NewVidLlmExtractPipelineStep(extractor ocr.LlmMediaExtractor, converter ocr.Video2AudioConverter, cfg *conf.CommonExtractingConfig) ExtractPipelineStep {
	return &VidLlmExtractPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 52,
			typ: []entity.MetadataType{entity.VideoMetadataType},
			key: VidLlmExtractKey,
		},
		extractor:     extractor,
		video2audio:   converter,
		separateAudio: cfg.SeparateAudio,
		slogger:       slog.With("service", "VidLlmExtractPipelineStep"),
	}
}
