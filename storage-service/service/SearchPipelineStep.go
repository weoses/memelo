package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
)

// PipelineAfterID is the pagination cursor returned by the search pipeline.
// SearcherName identifies which searcher to resume on the next call.
// SortKey is the Elasticsearch search_after cursor for that searcher.
type PipelineAfterID struct {
	SearcherName string
	SortKey      entity.ElasticSortKey
}

type SearchPipelineStep interface {
	GetIndex() int
	GetName() string
	Search(ctx context.Context, accountId uuid.UUID, query string, sortKey entity.ElasticSortKey, size int) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error)
}

type SearcherBase struct {
	Name  string
	Index int
}

func (b SearcherBase) GetName() string { return b.Name }
func (b SearcherBase) GetIndex() int   { return b.Index }
