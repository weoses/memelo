package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
)

type SearchPipelineStep interface {
	GetIndex() int
	GetName() string
	Search(ctx context.Context, accountId uuid.UUID, query string, afterId *int64, size *int) ([]*entity.ElasticImageMetaData, error)
}

type SearcherBase struct {
	Name  string
	Index int
}

func (b SearcherBase) GetName() string { return b.Name }
func (b SearcherBase) GetIndex() int   { return b.Index }
