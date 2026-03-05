package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

type CalcSizesPipelineStep struct {
	imageConverter ocr.ImageConveter
}

func (s *CalcSizesPipelineStep) GetPos() int {
	return 70
}

func (s *CalcSizesPipelineStep) Do(ctx context.Context, pCtx *ImageMetadataPipelineContext) error {
	if pCtx.ImageThumbnail == nil {
		return fmt.Errorf("error: image thumbnail can't be nil")
	}

	wRaw, hRaw, err := s.imageConverter.GetSize(ctx, pCtx.ImageRaw)
	if err != nil {
		return fmt.Errorf("error getting size of raw image: %w", err)
	}

	wThumb, hThumb, err := s.imageConverter.GetSize(ctx, pCtx.ImageThumbnail)
	if err != nil {
		return fmt.Errorf("error getting size of thumbnail: %w", err)
	}

	pCtx.ImageRawSize = entity.ElasticSizes{Width: wRaw, Height: hRaw}
	pCtx.ImageThumbnailSize = entity.ElasticSizes{Width: wThumb, Height: hThumb}
	return nil
}

func NewCalcSizesPipelineStep(imageConverter ocr.ImageConveter) ExtractPipelineStep {
	return &CalcSizesPipelineStep{imageConverter: imageConverter}
}
