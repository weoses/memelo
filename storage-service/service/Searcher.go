package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

type Searcher interface {
	GetIndex() int
	GetName() string
	Search(ctx context.Context, accountId uuid.UUID, query string, afterId *uuid.UUID, size *int) ([]*entity.ElasticImageMetaData, error)
}

// ===========================

type SearcherBase struct {
	name  string
	index int
}

func (b SearcherBase) GetName() string { return b.name }
func (b SearcherBase) GetIndex() int   { return b.index }

// ===========================

type AllSearcher struct {
	SearcherBase
	metadata MetadataStorageService
}

func (a AllSearcher) Search(ctx context.Context, accountId uuid.UUID, query string, afterId *uuid.UUID, size *int) ([]*entity.ElasticImageMetaData, error) {
	if query != "" {
		return make([]*entity.ElasticImageMetaData, 0), nil
	}

	matchedMetadataAll, err := a.metadata.SearchAll(
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

func NewAllSearcher(m MetadataStorageService) Searcher {
	return &AllSearcher{
		SearcherBase: SearcherBase{
			name:  "all_searcher",
			index: 40,
		},
		metadata: m,
	}
}

// ===========================

type SimpleSearcher struct {
	SearcherBase
	metadata MetadataStorageService
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

func NewSimpleSearcher(m MetadataStorageService) Searcher {
	return &SimpleSearcher{
		SearcherBase: SearcherBase{
			name:  "simple_searcher",
			index: 0,
		},
		metadata: m,
	}
}

// ===========================

type IdSearcher struct {
	SearcherBase
	metadata MetadataStorageService
}

func (s IdSearcher) Search(ctx context.Context, accountId uuid.UUID, query string, afterId *uuid.UUID, size *int) ([]*entity.ElasticImageMetaData, error) {
	if query == "" {
		return make([]*entity.ElasticImageMetaData, 0), nil
	}
	if afterId != nil {
		return make([]*entity.ElasticImageMetaData, 0), nil
	}

	idUuid, err := uuid.Parse(query)
	if err != nil {
		return make([]*entity.ElasticImageMetaData, 0), nil
	}

	matchedMetadataAll, err := s.metadata.GetById(
		ctx,
		accountId,
		idUuid,
	)

	if err != nil {
		return nil, fmt.Errorf("searcher %s failed: %w", s.GetName(), err)
	}

	if matchedMetadataAll == nil {
		return make([]*entity.ElasticImageMetaData, 0), nil
	}

	return []*entity.ElasticImageMetaData{matchedMetadataAll}, nil
}

func NewIdSearcher(m MetadataStorageService) Searcher {
	return &IdSearcher{
		SearcherBase: SearcherBase{
			name:  "id_searcher",
			index: 10,
		},
		metadata: m,
	}
}

// ===========================

type FuzzySearcher struct {
	SearcherBase
	metadata MetadataStorageService
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

func NewFuzzySearcher(m MetadataStorageService) Searcher {
	return &FuzzySearcher{
		SearcherBase: SearcherBase{
			name:  "fuzzy_searcher",
			index: 20,
		},
		metadata: m,
	}
}

// ===========================

type TextEmbeddingSearcher struct {
	SearcherBase
	metadata MetadataStorageService
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

	results, err := s.metadata.SearchByEmbeddingV1(ctx, accountId, embedding, count, false)
	if err != nil {
		return nil, fmt.Errorf("searcher %s: SearchByEmbeddingV1 failed: %w", s.GetName(), err)
	}

	return results, nil
}

func NewTextEmbeddingSearcher(m MetadataStorageService, e ocr.EmbeddingExtractor) Searcher {
	return &TextEmbeddingSearcher{
		SearcherBase: SearcherBase{
			name:  "text_embedding_searcher",
			index: 30,
		},
		metadata: m,
		embedder: e,
	}
}
