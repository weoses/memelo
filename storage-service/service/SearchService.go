package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/service/search_pipeline"
)

type SearchService interface {
	Search(ctx context.Context, accountId uuid.UUID, query string, afterId *uuid.UUID, size *int) ([]*entity.ElasticImageMetaData, error)
}

type SearchServiceImpl struct {
	searchers []search_pipeline.SearchPipelineStep
	slogger   *slog.Logger
}

func (m *SearchServiceImpl) Search(ctx context.Context, accountId uuid.UUID, query string, afterId *uuid.UUID, size *int) ([]*entity.ElasticImageMetaData, error) {
	elasticData := make([]*entity.ElasticImageMetaData, 0)
	for _, searcher := range m.searchers {
		searcherName := searcher.GetName()

		m.slogger.DebugContext(ctx, "Start searcher",
			"searcher", searcherName)

		data, err := searcher.Search(ctx, accountId, query, afterId, size)

		m.slogger.DebugContext(ctx, "End searcher",
			"searcher", searcherName,
			"results", len(data))

		if err != nil {
			return nil, fmt.Errorf("searcher %s failed: %w", searcherName, err)
		}

		if len(data) > 0 {
			elasticData = append(elasticData, data...)
			break
		}
	}
	return elasticData, nil
}

func NewSearchServiceImpl(searchers []search_pipeline.SearchPipelineStep, slogger *slog.Logger) SearchService {
	slices.SortFunc(searchers, func(a search_pipeline.SearchPipelineStep, b search_pipeline.SearchPipelineStep) int {
		return a.GetIndex() - b.GetIndex()
	})
	return &SearchServiceImpl{
		searchers: searchers,
		slogger:   slogger,
	}
}
