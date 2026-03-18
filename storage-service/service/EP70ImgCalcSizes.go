package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

type ImageCalcSizesPipelineStep struct {
	BasePipelineStep

	imageConverter ocr.ImageConveter
}

func (s *ImageCalcSizesPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	if pCtx.ImageThumbnail == nil {
		return fmt.Errorf("error: image thumbnail can't be nil")
	}

	wRaw, hRaw, err := s.imageConverter.GetSize(ctx, pCtx.ImageOriginalJpeg)
	if err != nil {
		return fmt.Errorf("error getting size of raw image: %w", err)
	}

	wThumb, hThumb, err := s.imageConverter.GetSize(ctx, pCtx.ImageThumbnail)
	if err != nil {
		return fmt.Errorf("error getting size of thumbnail: %w", err)
	}

	pCtx.ImageOriginalSize = entity.Sizes{Width: wRaw, Height: hRaw}
	pCtx.ImageThumbnailSize = entity.Sizes{Width: wThumb, Height: hThumb}
	return nil
}

func NewImageCalcSizesPipelineStep(imageConverter ocr.ImageConveter) ExtractPipelineStep {
	return &ImageCalcSizesPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 70,
			typ: []entity.MetadataType{entity.ImageMetadataType},
		},
		imageConverter: imageConverter}
}
