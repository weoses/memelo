package service

import (
	"context"

	"github.com/weoses/memelo/storage-service/storage"
)

type CalcTagsPipelineStep struct {
	tagStorage storage.ElasticTagStorage
}

func (s *CalcTagsPipelineStep) GetPos() int {
	return 80
}

func (s *CalcTagsPipelineStep) Do(ctx context.Context, pCtx *ImageMetadataPipelineContext) error {
	tags, err := s.tagStorage.SearchTagsByEmbedding(ctx, pCtx.AccountId, pCtx.ImageEmbedding, 0.8, 0.0)
	if err != nil {
		return err
	}
	pCtx.Tags = tags
	return nil
}

func NewCalcTagsPipelineStep(tagStorage storage.ElasticTagStorage) ExtractPipelineStep {
	return &CalcTagsPipelineStep{tagStorage: tagStorage}
}
