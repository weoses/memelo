package service

import (
	"context"
	"fmt"

	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

const ImgLlmExtractKey = "ImgLlmExtractPipelineStep"

type ImgLlmExtractPipelineStep struct {
	BasePipelineStep

	extractor ocr.LlmMediaExtractor
}

func (s *ImgLlmExtractPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	result, err := s.extractor.ProcessImage(ctx, pCtx.ImageOriginalJpeg)
	if err != nil {
		return fmt.Errorf("llm image extraction failed: %w", err)
	}
	pCtx.Result = &entity.Result{
		OnScreenText:    result.OnScreenText,
		AudioTranscript: result.AudioTranscript,
		AudioTrack:      result.AudioTrack,
		Caption:         result.Caption,
	}
	return nil
}

func NewImgLlmExtractPipelineStep(extractor ocr.LlmMediaExtractor) ExtractPipelineStep {
	return &ImgLlmExtractPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 52,
			typ: []entity.MetadataType{entity.ImageMetadataType},
			key: ImgLlmExtractKey,
		},
		extractor: extractor,
	}
}
