package api

import (
	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/storage-service/service"
)

func pipelineAfterIDFromProto(p *v1.PipelinePagination) *service.PipelineAfterID {
	if p == nil {
		return nil
	}
	return &service.PipelineAfterID{
		SearcherName: p.Searcher,
		SortKey:      p.SortingAfter,
	}
}

func pipelineAfterIDToProto(a *service.PipelineAfterID) *v1.PipelinePagination {
	if a == nil {
		return nil
	}
	return &v1.PipelinePagination{
		Searcher:     a.SearcherName,
		SortingAfter: a.SortKey,
	}
}
