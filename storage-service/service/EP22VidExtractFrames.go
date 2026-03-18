package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/common/helper"
	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

type VidExtractFramesPipelineStep struct {
	BasePipelineStep

	extractor      ocr.Video2FrameExtractor
	tmpDataService commonservice.TmpDataService
}

func (s *VidExtractFramesPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	if pCtx.VideoMp4 == nil {
		return nil
	}
	frames, err := s.extractor.ExtractFrames(ctx, pCtx.VideoMp4)
	if err != nil {
		return fmt.Errorf("cannot extract frames from video: %w", err)
	}

	framesS3Backed, err := helper.TransformSliceErr[temp.Data, temp.S3BackedData](
		frames,
		make([]temp.S3BackedData, len(frames)),
		func(data temp.Data) (temp.S3BackedData, error) {
			return s.tmpDataService.WrapData(ctx, data)
		})

	if err != nil {
		return fmt.Errorf("cannot transform frames to s3 backed data: %w", err)
	}

	pCtx.VideoFrames = framesS3Backed
	return nil
}

func NewVidExtractFramesPipelineStep(extractor ocr.Video2FrameExtractor, tmpDataService commonservice.TmpDataService) ExtractPipelineStep {
	return &VidExtractFramesPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 22,
			typ: []entity.MetadataType{entity.VideoMetadataType},
		},
		extractor:      extractor,
		tmpDataService: tmpDataService,
	}
}
