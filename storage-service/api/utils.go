package api

import (
	"encoding/json"
	"fmt"

	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/service"
)

func pipelineAfterIDFromProto(p *v1.PipelinePagination) *service.PipelineAfterID {
	if p == nil {
		return nil
	}
	sortKey := make(entity.ElasticSortKey, len(p.SortingAfter))
	for k, v := range p.SortingAfter {
		var val interface{}
		if err := json.Unmarshal([]byte(v), &val); err != nil {
			val = v
		}
		sortKey[k] = val
	}
	return &service.PipelineAfterID{
		SearcherName: p.Searcher,
		SortKey:      sortKey,
	}
}

func pipelineAfterIDToProto(a *service.PipelineAfterID) *v1.PipelinePagination {
	if a == nil {
		return nil
	}
	sortingAfter := make(map[string]string, len(a.SortKey))
	for k, v := range a.SortKey {
		b, err := json.Marshal(v)
		if err != nil {
			sortingAfter[k] = fmt.Sprintf("%v", v)
		} else {
			sortingAfter[k] = string(b)
		}
	}
	return &v1.PipelinePagination{
		Searcher:     a.SearcherName,
		SortingAfter: sortingAfter,
	}
}
