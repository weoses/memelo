package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/ocr"
)

type OcrImagePipelineStep struct {
	image2text ocr.TextExtractor
}

func (s *OcrImagePipelineStep) GetPos() int {
	return 50
}

func (s *OcrImagePipelineStep) Check(_ context.Context, steps *StepsToDo) (bool, error) {
	return steps.Ocr, nil
}

func (s *OcrImagePipelineStep) Do(ctx context.Context, pCtx *ImageMetadataPipelineContext) error {
	ocrResult, err := s.image2text.DoOcr(ctx, pCtx.ImageRaw)
	if err != nil {
		return fmt.Errorf("error ocring image: %w", err)
	}
	pCtx.ImageOcrResult = &ocrResult
	return nil
}

func NewOcrImagePipelineStep(image2text ocr.TextExtractor) ExtractPipelineStep {
	return &OcrImagePipelineStep{image2text: image2text}
}
