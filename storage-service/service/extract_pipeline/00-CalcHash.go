package extract_pipeline

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/storage-service/service"
)

type CalcHashPipelineStep struct{}

func (c *CalcHashPipelineStep) GetPos() int {
	return 0
}

func (c *CalcHashPipelineStep) Check(_ context.Context, steps *service.StepsToDo) (bool, error) {
	return steps.CalcHash, nil
}

func (c *CalcHashPipelineStep) Do(_ context.Context, pipelineContext *service.PipelineContext) error {
	hash, err := calcRawImageHash(pipelineContext.ImageRaw)
	if err != nil {
		return fmt.Errorf("create pipeline: error calculating hash: %w", err)
	}
	pipelineContext.ImageHash = &hash
	return nil
}

func calcRawImageHash(raw []byte) (string, error) {
	base64DataBuffer := bytes.NewBuffer(make([]byte, 0))
	encoder := base64.NewEncoder(base64.RawStdEncoding, base64DataBuffer)
	_, err := encoder.Write(raw)
	if err != nil {
		return "", fmt.Errorf("failed to write raw data to base64 encode buffer: %w", err)
	}
	hash := helper.CalcHash(base64DataBuffer.String())
	return hash, nil
}

func NewCalcHashPipelineStep() ExtractPipelineStep {
	return &CalcHashPipelineStep{}
}
