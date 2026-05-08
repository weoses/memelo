package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/entity"
)

const EncodeOriginalKey = "EncodeOriginalPipelineStep"

type EncodeOriginalPipelineStep struct {
	BasePipelineStep

	encoderService MediaEncoderService
}

func (s *EncodeOriginalPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	encoder := s.encoderService.GetEncoderByType(inputContext.Type)
	encoded, size, err := encoder.EncodeOriginal(ctx, inputContext.RawInput)
	if err != nil {
		return fmt.Errorf("EncodeOriginalPipelineStep: %w", err)
	}

	pCtx.OriginalSize = size
	if inputContext.Type == entity.ImageMetadataType {
		pCtx.ImageOriginalJpeg = encoded
	} else {
		pCtx.VideoMp4 = encoded
	}
	pCtx.StorageArtifacts = append(pCtx.StorageArtifacts, MetadataStorageArtifact{Type: SavedOriginal, Data: encoded})
	return nil
}

func NewEncodeOriginalPipelineStep(encoderService MediaEncoderService) ExtractPipelineStep {
	return &EncodeOriginalPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 20,
			typ: []entity.MetadataType{entity.ImageMetadataType, entity.VideoMetadataType},
			key: EncodeOriginalKey,
		},
		encoderService: encoderService,
	}
}
