package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/storage"
)

type IdSearcher struct {
	SearcherBase

	metadata storage.MetadataStorageService
}

func (s IdSearcher) Search(ctx context.Context, accountId uuid.UUID, query string, sortKey entity.ElasticSortKey, size int) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error) {
	if query == "" || sortKey != nil {
		return []*entity.ElasticImageMetaData{}, nil, nil
	}

	idUuid, err := uuid.Parse(query)
	if err != nil {
		return []*entity.ElasticImageMetaData{}, nil, nil
	}

	result, err := s.metadata.GetById(ctx, accountId, idUuid)
	if err != nil {
		return nil, nil, fmt.Errorf("searcher %s failed: %w", s.GetName(), err)
	}

	if result == nil {
		return []*entity.ElasticImageMetaData{}, nil, nil
	}

	return []*entity.ElasticImageMetaData{result}, nil, nil
}

func NewIdSearcher(m storage.MetadataStorageService) SearchPipelineStep {
	return &IdSearcher{
		SearcherBase: SearcherBase{Name: "id_searcher", Index: 0},
		metadata:     m,
	}
}
