package service

import (
	"context"
	"fmt"
	"log/slog"

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

func (s HybridSearcher) Search(ctx context.Context, accountId uuid.UUID, query string, metadataType *entity.MetadataType, sortKey entity.ElasticSortKey, size int) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error) {
	if query == "" {
		return []*entity.ElasticImageMetaData{}, nil, nil
	}
	s.slogger.InfoContext(ctx, "search", "query", query, "sortKey", sortKey, "size", size)

	embedding, err := s.embedder.GetTextEmbedding(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("searcher %s: GetTextEmbedding failed: %w", s.GetName(), err)
	}

	results, nextKey, err := s.metadata.SearchHybridOrderByScore(
		ctx, accountId, query, *embedding, s.searchConfig.Fuzziness, metadataType, sortKey, size,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("searcher %s failed: %w", s.GetName(), err)
	}

	return results, nextKey, nil
}

func NewHybridSearcher(m storage.MetadataStorageService, e ocr.LlmEmbeddingExtractor, cfg *conf.Config) SearchPipelineStep {
	name := "hybrid_searcher"
	return &HybridSearcher{
		SearcherBase: SearcherBase{Name: name, Index: 10, slogger: slog.With("service", name)},
		metadata:     m,
		embedder:     e,
		searchConfig: cfg.Search,
	}
}
