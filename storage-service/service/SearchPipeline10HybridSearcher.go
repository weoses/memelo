package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
	"github.com/weoses/memelo/storage-service/storage"
)

type HybridSearcher struct {
	SearcherBase

	metadata     storage.MetadataStorageService
	embedder     ocr.LlmEmbeddingExtractor
	searchConfig *conf.SearchConfig
}

func (s HybridSearcher) Search(ctx context.Context, accountId uuid.UUID, query string, sortKey entity.ElasticSortKey, size int) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error) {
	if query == "" {
		return []*entity.ElasticImageMetaData{}, nil, nil
	}

	embedding, err := s.embedder.GetTextEmbedding(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("searcher %s: GetTextEmbedding failed: %w", s.GetName(), err)
	}

	results, nextKey, err := s.metadata.SearchHybridOrderByScore(
		ctx, accountId, query, *embedding, s.searchConfig.Fuzziness, sortKey, size,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("searcher %s failed: %w", s.GetName(), err)
	}

	return results, nextKey, nil
}

func NewHybridSearcher(m storage.MetadataStorageService, e ocr.LlmEmbeddingExtractor, cfg *conf.Config) SearchPipelineStep {
	return &HybridSearcher{
		SearcherBase: SearcherBase{Name: "hybrid_searcher", Index: 10},
		metadata:     m,
		embedder:     e,
		searchConfig: cfg.Search,
	}
}
