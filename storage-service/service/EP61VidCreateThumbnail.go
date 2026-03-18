package service

import (
	"context"
	"fmt"

	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

type VidCreateThumbnailPipelineStep struct {
	BasePipelineStep

	imageConverter ocr.ImageConveter
	tmpDataService commonservice.TmpDataService
}

func (s *VidCreateThumbnailPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	if len(pCtx.VideoFrames) == 0 {
		return nil
	}
	thumb, err := s.imageConverter.MakeThumbnail(ctx, pCtx.VideoFrames[0])
	if err != nil {
		return fmt.Errorf("cannot create video thumbnail: %w", err)
	}

	s3WrappedThumb, err := s.tmpDataService.WrapData(ctx, thumb)
	if err != nil {
		return fmt.Errorf("cannot wrap data to s3 data: %w", err)
	}

	pCtx.VideoThumbnail = s3WrappedThumb
	pCtx.StorageArtifacts = append(pCtx.StorageArtifacts, MetadataStorageArtifact{Type: SavedThumb, Data: s3WrappedThumb})

	w, h, err := s.imageConverter.GetSize(ctx, thumb)
	if err != nil {
		return fmt.Errorf("cannot get video thumbnail size: %w", err)
	}
	pCtx.VideoThumbnailSizes = entity.Sizes{Width: w, Height: h}
	return nil
}

func NewVidCreateThumbnailPipelineStep(imageConverter ocr.ImageConveter, tmpDataService commonservice.TmpDataService) ExtractPipelineStep {
	return &VidCreateThumbnailPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 61,
			typ: []entity.MetadataType{entity.VideoMetadataType},
		},
		imageConverter: imageConverter,
		tmpDataService: tmpDataService,
	}
}
