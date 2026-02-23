package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
)

type Searcher interface {
	GetName() string
	Search(ctx context.Context, accountId uuid.UUID, query string, afterId *uuid.UUID, size *int) ([]*entity.ElasticImageMetaData, error)
}

// ===========================

type AllSearcher struct {
	metadata MetadataStorageService
}

func (a AllSearcher) GetName() string { return "all_searcher" }

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
		metadata: m,
	}
}

// ===========================

type SimpleSearcher struct {
	metadata MetadataStorageService
}

func (s SimpleSearcher) GetName() string { return "simple_searcher" }

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
		metadata: m,
	}
}

// ===========================

type IdSearcher struct {
	metadata MetadataStorageService
}

func (s IdSearcher) GetName() string { return "id_searcher" }

func (s IdSearcher) Search(ctx context.Context, accountId uuid.UUID, query string, afterId *uuid.UUID, size *int) ([]*entity.ElasticImageMetaData, error) {
	if query == "" {
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
		metadata: m,
	}
}
