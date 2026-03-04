package search_pipeline

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/service"
)

type AllSearcher struct {
	service.SearcherBase
	metadata service.MetadataStorageService
}

func (a AllSearcher) Search(ctx context.Context, accountId uuid.UUID, query string, afterId *uuid.UUID, size *int) ([]*entity.ElasticImageMetaData, error) {
	if query != "" {
		return make([]*entity.ElasticImageMetaData, 0), nil
	}

	matchedMetadataAll, err := a.metadata.SearchByAccountId(
		ctx,
		accountId,
		afterId,
		size,
	)

	if err != nil {
		return nil, fmt.Errorf("searcher %s failed: %w", a.GetName(), err)
	}
	return matchedMetadataAll, nil
}

func NewAllSearcher(m service.MetadataStorageService) service.Searcher {
	return &AllSearcher{
		SearcherBase: service.SearcherBase{Name: "all_searcher", Index: 40},
		metadata:     m,
	}
}
