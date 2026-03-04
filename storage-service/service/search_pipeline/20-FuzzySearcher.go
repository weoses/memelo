package search_pipeline

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/service"
)

type FuzzySearcher struct {
	service.SearcherBase
	metadata service.MetadataStorageService
}

func (s FuzzySearcher) Search(ctx context.Context, accountId uuid.UUID, query string, afterId *uuid.UUID, size *int) ([]*entity.ElasticImageMetaData, error) {
	if query == "" {
		return make([]*entity.ElasticImageMetaData, 0), nil
	}
	if afterId != nil {
		return make([]*entity.ElasticImageMetaData, 0), nil
	}

	matchedMetadataAll, err := s.metadata.SearchFuzzy(
		ctx,
		accountId,
		query,
		size,
	)

	if err != nil {
		return nil, fmt.Errorf("searcher %s failed: %w", s.GetName(), err)
	}

	if matchedMetadataAll == nil {
		return make([]*entity.ElasticImageMetaData, 0), nil
	}

	return matchedMetadataAll, nil
}

func NewFuzzySearcher(m service.MetadataStorageService) service.Searcher {
	return &FuzzySearcher{
		SearcherBase: service.SearcherBase{Name: "fuzzy_searcher", Index: 20},
		metadata:     m,
	}
}
