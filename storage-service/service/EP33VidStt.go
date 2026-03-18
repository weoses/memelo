package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

type VidSttPipelineStep struct {
	BasePipelineStep

	stt ocr.Audio2TextExtractor
}

func (s *VidSttPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	if pCtx.VideoAudio == nil {
		return nil
	}
	transcript, err := s.stt.Transcript(ctx, pCtx.VideoAudio)
	if err != nil {
		return fmt.Errorf("cannot transcribe video audio: %w", err)
	}
	pCtx.Transcription = transcript
	return nil
}

func NewVidSttPipelineStep(stt ocr.Audio2TextExtractor) ExtractPipelineStep {
	return &VidSttPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 33,
			typ: []entity.MetadataType{entity.VideoMetadataType},
		},
		stt: stt,
	}
}
