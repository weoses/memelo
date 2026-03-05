package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/storage"
)

type SimpleSearcher struct {
	SearcherBase
	metadata storage.MetadataStorageService
}

func (s SimpleSearcher) Search(ctx context.Context, accountId uuid.UUID, query string, afterId *uuid.UUID, size *int) ([]*entity.ElasticImageMetaData, error) {
	if query == "" {
		return make([]*entity.ElasticImageMetaData, 0), nil
	}

	matchedMetadataAll, err := s.metadata.SearchSimple(
		ctx,
		accountId,
		query,
		afterId,
		size,
	)

	if err != nil {
		return nil, fmt.Errorf("searcher %s failed: %w", s.GetName(), err)
	}
	return matchedMetadataAll, nil
}

func NewSimpleSearcher(m storage.MetadataStorageService) SearchPipelineStep {
	return &SimpleSearcher{
		SearcherBase: SearcherBase{Name: "simple_searcher", Index: 0},
		metadata:     m,
	}
}
