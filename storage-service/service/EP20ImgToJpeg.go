package service //nolint:dupl

import (
	"context"
	"fmt"

	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

type ImageToJpegPipelineStep struct {
	BasePipelineStep

	imageConverter ocr.ImageConveter
	tmpDataService commonservice.TmpDataService
}

func (s *ImageToJpegPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	imgJpeg, err := s.imageConverter.Convert2Jpeg(ctx, inputContext.RawInput)
	if err != nil {
		return fmt.Errorf("ImageToJpegPipelineStep: cannot process original image: %w", err)
	}

	s3WrappedData, err := s.tmpDataService.WrapData(ctx, "image/jpeg", imgJpeg)
	if err != nil {
		return fmt.Errorf("ImageToJpegPipelineStep: cannot wrap s3 for image: %w", err)
	}

	pCtx.ImageOriginalJpeg = s3WrappedData
	pCtx.StorageArtifacts = append(pCtx.StorageArtifacts, MetadataStorageArtifact{Type: SavedOriginal, Data: s3WrappedData})
	return nil
}

func NewImageToJpegPipelineStep(imageConverter ocr.ImageConveter, tmpDataService commonservice.TmpDataService) ExtractPipelineStep {
	return &ImageToJpegPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 20,
			typ: []entity.MetadataType{entity.ImageMetadataType},
		},
		imageConverter: imageConverter,
		tmpDataService: tmpDataService,
	}
}
