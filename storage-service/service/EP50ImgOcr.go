package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

type ImageOcrImagePipelineStep struct {
	BasePipelineStep

	image2text ocr.Image2TextExtractor
}

func (s *ImageOcrImagePipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	ocrResult, err := s.image2text.DoOcr(ctx, pCtx.ImageOriginalJpeg)
	if err != nil {
		return fmt.Errorf("error ocring image: %w", err)
	}
	if pCtx.Result == nil {
		pCtx.Result = &entity.Result{}
	}
	pCtx.Result.OnScreenText = ocrResult
	return nil
}

func NewImageOcrImagePipelineStep(image2text ocr.Image2TextExtractor) ExtractPipelineStep {
	return &ImageOcrImagePipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 50,
			typ: []entity.MetadataType{entity.ImageMetadataType},
		},
		image2text: image2text}
}
