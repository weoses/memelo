package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/storage"
)

type IdSearcher struct {
	SearcherBase

	metadata storage.MetadataStorageService
}

func (s IdSearcher) Search(ctx context.Context, accountId uuid.UUID, query string, _ *entity.MetadataType, sortKey entity.ElasticSortKey, size int) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error) {
	if query == "" || sortKey != nil {
		return []*entity.ElasticImageMetaData{}, nil, nil
	}
	s.slogger.InfoContext(ctx, "search", "query", query, "sortKey", sortKey, "size", size)

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
	name := "id_searcher"
	return &IdSearcher{
		SearcherBase: SearcherBase{Name: name, Index: 0, slogger: slog.With("service", name)},
		metadata:     m,
	}
}
