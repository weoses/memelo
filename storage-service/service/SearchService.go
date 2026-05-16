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
	AfterID *PipelineAfterID
	Result  []*entity.ElasticImageMetaData
}

type SearchService interface {
	Search(ctx context.Context, accountId uuid.UUID, query string, metadataType *entity.MetadataType, afterId *PipelineAfterID, size int) (*SearchServiceResponse, error)
}

type SearchServiceImpl struct {
	searchers []SearchPipelineStep
	slogger   *slog.Logger
}

func (m *SearchServiceImpl) Search(
	ctx context.Context,
	accountId uuid.UUID,
	query string,
	metadataType *entity.MetadataType,
	afterId *PipelineAfterID,
	size int,
) (*SearchServiceResponse, error) {
	var encounteredLastSearcher = true
	var skipUntilName string
	var resumeSortKey entity.ElasticSortKey

	if afterId != nil {
		encounteredLastSearcher = false
		skipUntilName = afterId.SearcherName
		resumeSortKey = afterId.SortKey
	}

	for _, searcher := range m.searchers {
		searcherName := searcher.GetName()

		if !encounteredLastSearcher && searcherName != skipUntilName {
			continue
		}

		var sortKey entity.ElasticSortKey
		if !encounteredLastSearcher && searcherName == skipUntilName {
			sortKey = resumeSortKey
			encounteredLastSearcher = true
		}

		m.slogger.DebugContext(ctx, "Start searcher",
			"searcher", searcherName)

		data, returnedKey, err := searcher.Search(ctx, accountId, query, metadataType, sortKey, size)

		m.slogger.DebugContext(ctx, "End searcher",
			"searcher", searcherName,
			"results", len(data))

		if err != nil {
			return nil, fmt.Errorf("searcher %s failed: %w", searcherName, err)
		}

		if len(data) > 0 {
			var nextAfterID *PipelineAfterID
			if returnedKey != nil {
				nextAfterID = &PipelineAfterID{
					SearcherName: searcherName,
					SortKey:      returnedKey,
				}
			}
			return &SearchServiceResponse{
				AfterID: nextAfterID,
				Result:  data,
			}, nil
		}
	}

	return &SearchServiceResponse{
		AfterID: nil,
		Result:  make([]*entity.ElasticImageMetaData, 0),
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
