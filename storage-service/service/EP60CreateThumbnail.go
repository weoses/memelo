package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/entity"
)

type CreateThumbnailPipelineStep struct {
	BasePipelineStep
	encoderService MediaEncoderService
}

func (s *CreateThumbnailPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	var source temp.S3BackedData
	if inputContext.Type == entity.ImageMetadataType {
		source = pCtx.ImageOriginalJpeg
	} else {
		source = pCtx.VideoMp4
	}
	if source == nil {
		return nil
	}

	encoder := s.encoderService.GetEncoderByType(inputContext.Type)
	thumb, size, err := encoder.EncodeThumbnail(ctx, source)
	if err != nil {
		return fmt.Errorf("CreateThumbnailPipelineStep: %w", err)
	}

	pCtx.ImageThumbnail = thumb
	pCtx.ThumbnailSize = size
	pCtx.StorageArtifacts = append(pCtx.StorageArtifacts, MetadataStorageArtifact{Type: SavedThumb, Data: thumb})
	return nil
}

func NewCreateThumbnailPipelineStep(encoderService MediaEncoderService) ExtractPipelineStep {
	return &CreateThumbnailPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 60,
			typ: []entity.MetadataType{entity.ImageMetadataType, entity.VideoMetadataType},
		},
		encoderService: encoderService,
	}
}
