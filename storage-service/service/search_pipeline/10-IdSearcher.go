package search_pipeline

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/service"
)

type IdSearcher struct {
	SearcherBase
	metadata service.MetadataStorageService
}

func (s IdSearcher) Search(ctx context.Context, accountId uuid.UUID, query string, afterId *uuid.UUID, size *int) ([]*entity.ElasticImageMetaData, error) {
	if query == "" {
		return make([]*entity.ElasticImageMetaData, 0), nil
	}
	if afterId != nil {
		return make([]*entity.ElasticImageMetaData, 0), nil
	}

	idUuid, err := uuid.Parse(query)
	if err != nil {
		return make([]*entity.ElasticImageMetaData, 0), nil
	}

	matchedMetadataAll, err := s.metadata.GetById(
		ctx,
		accountId,
		idUuid,
	)

	if err != nil {
		return nil, fmt.Errorf("searcher %s failed: %w", s.GetName(), err)
	}

	if matchedMetadataAll == nil {
		return make([]*entity.ElasticImageMetaData, 0), nil
	}

	return []*entity.ElasticImageMetaData{matchedMetadataAll}, nil
}

func NewIdSearcher(m service.MetadataStorageService) SearchPipelineStep {
	return &IdSearcher{
		SearcherBase: SearcherBase{Name: "id_searcher", Index: 10},
		metadata:     m,
	}
}
