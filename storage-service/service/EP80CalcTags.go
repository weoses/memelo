package service

import (
	"context"

	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/storage"
)

const CalcTagsKey = "CalcTagsPipelineStep"

type CalcTagsPipelineStep struct {
	BasePipelineStep

	tagStorage storage.ElasticTagStorage
}

func (s *CalcTagsPipelineStep) Do(ctx context.Context, inputContext MetadataInputContext, pCtx *MetadataPipelineContext) error {
	tags := make([]entity.ElasticTag, 0)
	for i := range len(pCtx.Embedding) {
		tagsChunk, err := s.tagStorage.SearchTagsByEmbedding(ctx, inputContext.AccountId, pCtx.Embedding[i], 0.8, 0.0)
		if err != nil {
			return err
		}
		tags = append(tags, tagsChunk...)
	}
	pCtx.Tags = tags
	return nil
}

func NewCalcTagsPipelineStep(tagStorage storage.ElasticTagStorage) ExtractPipelineStep {
	return &CalcTagsPipelineStep{
		BasePipelineStep: BasePipelineStep{
			pos: 80,
			typ: []entity.MetadataType{entity.ImageMetadataType, entity.VideoMetadataType},
			key: CalcTagsKey,
		},
		tagStorage: tagStorage,
	}
}
