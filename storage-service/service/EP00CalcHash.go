package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"

	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/entity"
)

const CalcHashKey = "CalcHashPipelineStep"

type CalcHashPipelineStep struct {
	BasePipelineStep
}

func (c *CalcHashPipelineStep) Do(_ context.Context, inputContext MetadataInputContext, pipelineContext *MetadataPipelineContext) error {
	if pipelineContext.Hash != "" {
		return nil
	}
	hash, err := calcRawHash(inputContext.RawInput)
	if err != nil {
		return fmt.Errorf("create pipeline: error calculating hash: %w", err)
	}
	pipelineContext.Hash = hash
	return nil
}

func calcRawHash(raw temp.Data) (string, error) {
	reader, err := raw.Reader()
	if err != nil {
		return "", fmt.Errorf("failed to read incoming temp %w", err)
	}
	defer helper.QuietClose(reader, slog.With("calcRawHash"))

	hasher := md5.New()
	_, err = io.Copy(hasher, reader)
	if err != nil {
		return "", fmt.Errorf("failed to write raw temp to hasher: %w", err)
	}

	byteHash := hasher.Sum(nil)
	return hex.EncodeToString(byteHash), nil
}

func NewCalcHashPipelineStep() ExtractPipelineStep {
	return &CalcHashPipelineStep{
		BasePipelineStep{
			pos: 0,
			typ: []entity.MetadataType{entity.ImageMetadataType, entity.VideoMetadataType},
			key: CalcHashKey,
		},
	}
}
