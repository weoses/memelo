package service

import (
	"context"
	"fmt"

	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

type VidExtractAudioPipelineStep struct {
	BasePipelineStep

	extractor      ocr.Video2AudioExtractor
	tmpDataService commonservice.TmpDataService
}

func (s *VidExtractAudioPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	if pCtx.VideoMp4 == nil {
		return nil
	}
	audio, err := s.extractor.ExtractAudio(ctx, pCtx.VideoMp4)
	if err != nil {
		return fmt.Errorf("cannot extract audio from video: %w", err)
	}

	s3WrappedAudio, err := s.tmpDataService.WrapData(ctx, audio)
	if err != nil {
		return fmt.Errorf("cannot wrap s3 for image: %w", err)
	}

	pCtx.VideoAudio = s3WrappedAudio
	return nil
}

func NewVidExtractAudioPipelineStep(extractor ocr.Video2AudioExtractor, tmpDataService commonservice.TmpDataService) ExtractPipelineStep {
	return &VidExtractAudioPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 23,
			typ: []entity.MetadataType{entity.VideoMetadataType},
		},
		extractor:      extractor,
		tmpDataService: tmpDataService,
	}
}
