package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/storage"
)

type AllSearcher struct {
	SearcherBase

	metadata storage.MetadataStorageService
}

func (a AllSearcher) Search(ctx context.Context, accountId uuid.UUID, query string, sortKey entity.ElasticSortKey, size int) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error) {
	if query != "" {
		return []*entity.ElasticImageMetaData{}, nil, nil
	}

	results, nextKey, err := a.metadata.GetByAccountIdOrderByCreated(ctx, accountId, sortKey, size)
	if err != nil {
		return nil, nil, fmt.Errorf("searcher %s failed: %w", a.GetName(), err)
	}

	return results, nextKey, nil
}

func NewAllSearcher(m storage.MetadataStorageService) SearchPipelineStep {
	return &AllSearcher{
		SearcherBase: SearcherBase{Name: "all_searcher", Index: 20},
		metadata:     m,
	}
}
