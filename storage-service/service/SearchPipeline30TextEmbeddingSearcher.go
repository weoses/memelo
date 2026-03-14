package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
	"github.com/weoses/memelo/storage-service/storage"
)

type TextEmbeddingSearcher struct {
	SearcherBase

	metadata storage.MetadataStorageService
	embedder ocr.EmbeddingExtractor
}

func (s TextEmbeddingSearcher) Search(ctx context.Context, accountId uuid.UUID, query string, afterId *uuid.UUID, size *int) ([]*entity.ElasticImageMetaData, error) {
	if query == "" {
		return make([]*entity.ElasticImageMetaData, 0), nil
	}
	if afterId != nil {
		return make([]*entity.ElasticImageMetaData, 0), nil
	}

	embedding, err := s.embedder.GetTextEmbeddingV1(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("searcher %s: GetTextEmbeddingV1 failed: %w", s.GetName(), err)
	}

	count := 5

	results, err := s.metadata.SearchByEmbeddingV1(ctx, accountId, *embedding, count, false)
	if err != nil {
		return nil, fmt.Errorf("searcher %s: SearchByEmbeddingV1 failed: %w", s.GetName(), err)
	}

	return results, nil
}

func NewTextEmbeddingSearcher(m storage.MetadataStorageService, e ocr.EmbeddingExtractor) SearchPipelineStep {
	return &TextEmbeddingSearcher{
		SearcherBase: SearcherBase{Name: "text_embedding_searcher", Index: 30},
		metadata:     m,
		embedder:     e,
	}
}
