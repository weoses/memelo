package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/ocr"
)

type ToJpegPipelineStep struct {
	imageConverter ocr.ImageConveter
}

func (s *ToJpegPipelineStep) GetPos() int {
	return 20
}

func (s *ToJpegPipelineStep) Do(ctx context.Context, pCtx *ImageMetadataPipelineContext) error {
	imgJpeg, err := s.imageConverter.ProcessOriginalImage(ctx, pCtx.ImageRaw)
	if err != nil {
		return fmt.Errorf("cannot process original image: %w", err)
	}
	pCtx.ImageRaw = imgJpeg
	return nil
}

func NewToJpegPipelineStep(imageConverter ocr.ImageConveter) ExtractPipelineStep {
	return &ToJpegPipelineStep{imageConverter: imageConverter}
}
