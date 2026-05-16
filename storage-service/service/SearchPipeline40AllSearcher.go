package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/storage"
)

type AllSearcher struct {
	SearcherBase

	metadata storage.MetadataStorageService
}

func (a AllSearcher) Search(ctx context.Context, accountId uuid.UUID, query string, metadataType *entity.MetadataType, sortKey entity.ElasticSortKey, size int) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error) {
	if query != "" {
		return []*entity.ElasticImageMetaData{}, nil, nil
	}
	a.slogger.InfoContext(ctx, "search", "query", query, "sortKey", sortKey, "size", size)

	results, nextKey, err := a.metadata.GetByAccountIdOrderByCreated(ctx, accountId, metadataType, sortKey, size)
	if err != nil {
		return nil, nil, fmt.Errorf("searcher %s failed: %w", a.GetName(), err)
	}

	return results, nextKey, nil
}

func NewAllSearcher(m storage.MetadataStorageService) SearchPipelineStep {
	name := "all_searcher"
	return &AllSearcher{
		SearcherBase: SearcherBase{Name: name, Index: 20, slogger: slog.With("service", name)},
		metadata:     m,
	}
}
