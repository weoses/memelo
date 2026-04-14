package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

type VidLlmExtractPipelineStep struct {
	BasePipelineStep

	extractor ocr.LlmMediaExtractor
}

func (s *VidLlmExtractPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	if pCtx.VideoMp4 == nil {
		return nil
	}
	result, err := s.extractor.ProcessVideo(ctx, pCtx.VideoMp4)
	if err != nil {
		return fmt.Errorf("llm video extraction failed: %w", err)
	}
	pCtx.Result = &entity.Result{
		OnScreenText:    result.OnScreenText,
		AudioTranscript: result.AudioTranscript,
		AudioTrack:      result.AudioTrack,
		Caption:         result.Caption,
	}
	return nil
}

func NewVidLlmExtractPipelineStep(extractor ocr.LlmMediaExtractor) ExtractPipelineStep {
	return &VidLlmExtractPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 52,
			typ: []entity.MetadataType{entity.VideoMetadataType},
		},
		extractor: extractor,
	}
}
