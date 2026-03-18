package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
)

type SearchServiceResponse struct {
	SearcherName string
	Result       []*entity.ElasticImageMetaData
}

type SearchService interface {
	Search(ctx context.Context, accountId uuid.UUID, query string, afterId *uuid.UUID, size *int) (*SearchServiceResponse, error)
}

type SearchServiceImpl struct {
	searchers []SearchPipelineStep
	slogger   *slog.Logger
}

func (m *SearchServiceImpl) Search(
	ctx context.Context,
	accountId uuid.UUID,
	query string,
	afterId *uuid.UUID,
	size *int,
) (*SearchServiceResponse, error) {
	selectedSearcherName := ""
	selectedElasticData := make([]*entity.ElasticImageMetaData, 0)
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
			selectedSearcherName = searcherName
			selectedElasticData = append(selectedElasticData, data...)
			break
		}
	}
	return &SearchServiceResponse{
		SearcherName: selectedSearcherName,
		Result:       selectedElasticData,
	}, nil
}

func NewSearchServiceImpl(searchers []SearchPipelineStep) SearchService {
	slices.SortFunc(searchers, func(a SearchPipelineStep, b SearchPipelineStep) int {
		return a.GetIndex() - b.GetIndex()
	})
	return &SearchServiceImpl{
		searchers: searchers,
		slogger:   slog.With("service", "SearchService"),
	}
}
