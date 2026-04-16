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

type TextEmbeddingSearcher struct {
	SearcherBase

	metadata     storage.MetadataStorageService
	embedder     ocr.LlmEmbeddingExtractor
	searchConfig *conf.SearchConfig
}

func (s TextEmbeddingSearcher) Search(ctx context.Context, accountId uuid.UUID, query string, afterId *int64, size *int) ([]*entity.ElasticImageMetaData, error) {
	if query == "" {
		return make([]*entity.ElasticImageMetaData, 0), nil
	}
	if afterId != nil {
		return make([]*entity.ElasticImageMetaData, 0), nil
	}

	embedding, err := s.embedder.GetTextEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("searcher %s: GetTextEmbedding failed: %w", s.GetName(), err)
	}

	results, err := s.metadata.SearchByEmbeddingV1(ctx, accountId, *embedding, size, s.searchConfig.SemanticTextSearchThreshold)
	if err != nil {
		return nil, fmt.Errorf("searcher %s: SearchByEmbeddingV1 failed: %w", s.GetName(), err)
	}

	return results, nil
}

func NewTextEmbeddingSearcher(m storage.MetadataStorageService, e ocr.LlmEmbeddingExtractor, cfg *conf.Config) SearchPipelineStep {
	return &TextEmbeddingSearcher{
		SearcherBase: SearcherBase{Name: "text_embedding_searcher", Index: 30},
		metadata:     m,
		embedder:     e,
		searchConfig: cfg.Search,
	}
}
