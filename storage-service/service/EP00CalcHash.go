package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"

	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/entity"
)

type CalcHashPipelineStep struct {
	BasePipelineStep
}

func (c *CalcHashPipelineStep) Do(_ context.Context, inputContext MetadataInputContext, pipelineContext *MetadataPipelineContext) error {
	hash, err := calcRawImageHash(inputContext.RawInput)
	if err != nil {
		return fmt.Errorf("create pipeline: error calculating hash: %w", err)
	}
	pipelineContext.Hash = hash
	return nil
}

func calcRawImageHash(raw temp.Data) (string, error) {
	base64DataBuffer := bytes.NewBuffer(make([]byte, 0))
	encoder := base64.NewEncoder(base64.RawStdEncoding, base64DataBuffer)
	reader, err := raw.Reader()
	if err != nil {
		return "", fmt.Errorf("failed to read incoming temp %w", err)
	}
	defer helper.QuietClose(reader, slog.With("calcRawImageHash"))
	_, err = io.Copy(encoder, reader)
	if err != nil {
		return "", fmt.Errorf("failed to write raw temp to base64 encode buffer: %w", err)
	}
	hash := helper.CalcHash(base64DataBuffer.String())
	return hash, nil
}

func NewCalcHashPipelineStep() ExtractPipelineStep {
	return &CalcHashPipelineStep{
		BasePipelineStep{
			pos: 0,
			typ: []entity.MetadataType{entity.ImageMetadataType, entity.VideoMetadataType},
		},
	}
}
