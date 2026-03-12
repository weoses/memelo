package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/ocr"
)

type CreateThumbnailPipelineStep struct {
	imageConverter ocr.ImageConveter
}

func (s *CreateThumbnailPipelineStep) GetPos() int {
	return 60
}

func (s *CreateThumbnailPipelineStep) Do(ctx context.Context, pCtx *ImageMetadataPipelineContext) error {
	imgThumb, err := s.imageConverter.MakeThumbnail(ctx, pCtx.ImageRaw)
	if err != nil {
		return fmt.Errorf("cannot create thumbnail: %w", err)
	}
	pCtx.ImageThumbnail = imgThumb
	return nil
}

func NewCreateThumbnailPipelineStep(imageConverter ocr.ImageConveter) ExtractPipelineStep {
	return &CreateThumbnailPipelineStep{imageConverter: imageConverter}
}
