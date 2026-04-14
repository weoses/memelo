package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/agnivade/levenshtein"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

type VidOcrFramesPipelineStep struct {
	BasePipelineStep

	image2text ocr.Image2TextExtractor
}

func (s *VidOcrFramesPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	if len(pCtx.VideoFrames) == 0 {
		return nil
	}
	if pCtx.Result == nil {
		pCtx.Result = &entity.Result{}
	}

	var results []string
	var errors []error

	waitGroup := &sync.WaitGroup{}
	for _, frame := range pCtx.VideoFrames {
		resultsPtr := &results
		errorsPtr := &errors

		waitGroup.Go(func() {
			text, err := s.image2text.DoOcr(ctx, frame)
			if err != nil {
				*errorsPtr = append(*errorsPtr, fmt.Errorf("cannot ocr video frame: %w", err))
				return
			}

			if text != "" {
				*resultsPtr = append(*resultsPtr, text)
			}
		})
	}
	waitGroup.Wait()
	if len(errors) > 0 {
		return fmt.Errorf("cannot ocr video frames: %w", errors[0])
	}

	var mergedResults []string

	for _, result := range results {
		duplicate := false
		duplicateIdx := -1

		for j, prevResult := range mergedResults {
			length := len(result)
			distance := levenshtein.ComputeDistance(prevResult, result)
			percentChange := math.Abs(float64(distance) / float64(length))
			if percentChange < 0.35 {
				duplicate = true
				duplicateIdx = j
				break
			}
		}

		if duplicate {
			duplicateItem := mergedResults[duplicateIdx]
			if len(duplicateItem) < len(result) {
				mergedResults[duplicateIdx] = result
			}
		} else {
			mergedResults = append(mergedResults, result)
		}
	}

	if len(mergedResults) > 0 {
		joined := strings.Join(mergedResults, "\n")
		pCtx.Result.OnScreenText = joined
	}

	return nil
}

func NewVidOcrFramesPipelineStep(image2text ocr.Image2TextExtractor) ExtractPipelineStep {
	return &VidOcrFramesPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 51,
			typ: []entity.MetadataType{entity.VideoMetadataType},
		},
		image2text: image2text,
	}
}
