package service //nolint:dupl

import (
	"context"
	"fmt"

	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

type VidToMp4PipelineStep struct {
	BasePipelineStep

	converter      ocr.Video2Mp4Converter
	tmpDataService commonservice.TmpDataService
}

func (s *VidToMp4PipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	resultMp4, err := s.converter.ConvertToMp4(ctx, inputContext.RawInput)
	if err != nil {
		return fmt.Errorf("VidToMp4PipelineStep: cannot convert video to mp4: %w", err)
	}

	s3WrappedMp4, err := s.tmpDataService.WrapData(ctx, resultMp4)
	if err != nil {
		return fmt.Errorf("VidToMp4PipelineStep: cannot wrap s3 for image: %w", err)
	}

	pCtx.VideoMp4 = s3WrappedMp4
	pCtx.StorageArtifacts = append(pCtx.StorageArtifacts, MetadataStorageArtifact{Type: SavedOriginal, Data: s3WrappedMp4})
	return nil
}

func NewVidToMp4PipelineStep(converter ocr.Video2Mp4Converter, tmpDataService commonservice.TmpDataService) ExtractPipelineStep {
	return &VidToMp4PipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 21,
			typ: []entity.MetadataType{entity.VideoMetadataType},
		},
		converter:      converter,
		tmpDataService: tmpDataService,
	}
}
