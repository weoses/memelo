package service

import (
	"context"
	"fmt"

	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

type ImageCreateThumbnailPipelineStep struct {
	BasePipelineStep

	imageConverter ocr.ImageConveter
	tmpDataService commonservice.TmpDataService
}

func (s *ImageCreateThumbnailPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	imgThumb, err := s.imageConverter.MakeThumbnail(ctx, pCtx.ImageOriginalJpeg)
	if err != nil {
		return fmt.Errorf("cannot create thumbnail: %w", err)
	}

	s3WrappedImgThumb, err := s.tmpDataService.WrapData(ctx, imgThumb)
	if err != nil {
		return fmt.Errorf("cannot wrap data to s3 data: %w", err)
	}

	pCtx.ImageThumbnail = s3WrappedImgThumb
	pCtx.StorageArtifacts = append(pCtx.StorageArtifacts, MetadataStorageArtifact{Type: SavedThumb, Data: s3WrappedImgThumb})
	return nil
}

func NewImageCreateThumbnailPipelineStep(imageConverter ocr.ImageConveter, tmpDataService commonservice.TmpDataService) ExtractPipelineStep {
	return &ImageCreateThumbnailPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 60,
			typ: []entity.MetadataType{entity.ImageMetadataType},
		},
		imageConverter: imageConverter,
		tmpDataService: tmpDataService,
	}
}
