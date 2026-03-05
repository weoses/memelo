package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"time"

	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/refresh"
	"github.com/gdexlab/go-render/render"
	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/entity"
)

type ElasticTagStorage interface {
	SaveTag(ctx context.Context, tag entity.ElasticTag) error
	ListTag(ctx context.Context, queryName *string, queryDescription *string) ([]entity.ElasticTag, error)
	DeleteTag(ctx context.Context, id uuid.UUID) error
	SearchTagsByEmbedding(ctx context.Context, tag entity.ElasticEmbeddingV1, percentileMatch float32, threshold float32) ([]entity.ElasticTag, error)
}

type ElasticTagStorageImpl struct {
	client  *elasticsearch8.TypedClient
	index   string
	slogger *slog.Logger
}

func NewElasticTagStorage(config *conf.ElasticTagConfig) (ElasticTagStorage, error) {
	es8, _ := elasticsearch8.NewTypedClient(*config.Elastic)
	logger := slog.With("service", "ElasticTagStorage")
	indexExists, err := es8.Indices.
		Exists(config.Index).
		Do(context.Background())

	if err != nil {
		return nil, fmt.Errorf("failed to check if elastic index exists: %w", err)
	}

	if !indexExists {
		responseCreate, err := es8.Indices.
			Create(config.Index).
			Do(context.Background())

		logger.InfoContext(context.Background(), "Elastic create index",
			"index", config.Index,
			"response", render.Render(responseCreate),
			"error", err)
	}

	indexTypeMapping := types.NewTypeMapping()
	indexTypeMapping.Properties["Created"] = types.NewLongNumberProperty()
	indexTypeMapping.Properties["Updated"] = types.NewLongNumberProperty()
	indexTypeMapping.Properties["Tag"] = types.NewKeywordProperty()

	denseProp := types.NewDenseVectorProperty()
	denseProp.Index = helper.Addr(true)
	denseProp.Dims = helper.Addr(config.EmbeddingV1Dimensions)
	denseProp.Similarity = helper.Addr("cosine")
	indexTypeMapping.Properties["EmbeddingV1.Data"] = denseProp

	responseMapping, err := es8.Indices.PutMapping(config.Index).
		Properties(indexTypeMapping.Properties).
		Do(context.Background())

	logger.InfoContext(context.Background(), "Elastic create mapping index",
		"response", render.Render(responseMapping),
		"error", err)

	return &ElasticTagStorageImpl{
		client:  es8,
		index:   config.Index,
		slogger: logger,
	}, nil
}

func (s *ElasticTagStorageImpl) SaveTag(ctx context.Context, tag entity.ElasticTag) error {
	tag.Updated = time.Now().UnixMicro()
	if tag.Created == 0 {
		tag.Created = tag.Updated
	}

	response, err := s.client.
		Index(s.index).
		Document(tag).
		Id(tag.Id.String()).
		Refresh(refresh.True).
		Do(ctx)

	if err != nil {
		return fmt.Errorf("save tag document error: id=%s : %w", tag.Id, err)
	}

	s.slogger.InfoContext(ctx, "SaveTag", "id", tag.Id)
	s.slogger.DebugContext(ctx, "SaveTag details", "id", tag.Id, "response", render.Render(response))
	return nil
}

func (s *ElasticTagStorageImpl) ListTag(ctx context.Context, queryName *string, queryDescription *string) ([]entity.ElasticTag, error) {
	q := types.NewQuery()

	if queryName == nil && queryDescription == nil {
		q.MatchAll = types.NewMatchAllQuery()
	} else {
		q.Bool = types.NewBoolQuery()
		musts := make([]types.Query, 0)

		if queryName != nil {
			nameQ := types.NewQuery()
			nameQ.Match = map[string]types.MatchQuery{
				"Tag": {Query: *queryName},
			}
			musts = append(musts, *nameQ)
		}

		if queryDescription != nil {
			descQ := types.NewQuery()
			descQ.Match = map[string]types.MatchQuery{
				"Description": {Query: *queryDescription},
			}
			musts = append(musts, *descQ)
		}

		q.Bool.Must = musts
	}

	result, err := s.client.Search().
		Index(s.index).
		Query(q).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("list tags query failed: %w", err)
	}

	entities := make([]entity.ElasticTag, len(result.Hits.Hits))
	for i, hit := range result.Hits.Hits {
		var entity entity.ElasticTag
		if err := json.Unmarshal(hit.Source_, &entity); err != nil {
			return nil, fmt.Errorf("list tags unmarshal failed at index %d: %w", i, err)
		}
		entities[i] = entity
	}

	return entities, nil
}

func (s *ElasticTagStorageImpl) DeleteTag(ctx context.Context, id uuid.UUID) error {
	q := types.NewQuery()
	q.Ids = types.NewIdsQuery()
	q.Ids.Values = []string{id.String()}

	result, err := s.client.DeleteByQuery(s.index).
		Query(q).
		Refresh(true).
		Do(ctx)

	if err != nil {
		return fmt.Errorf("delete tag query failed: id=%s : %w", id, err)
	}

	s.slogger.InfoContext(ctx, "DeleteTag", "id", id, "deleted", result.Deleted)
	return nil
}

func (s *ElasticTagStorageImpl) SearchTagsByEmbedding(ctx context.Context, tag entity.ElasticEmbeddingV1, percentileMatch float32, threshold float32) ([]entity.ElasticTag, error) {
	script := types.NewScript()
	script.Source = helper.Addr("cosineSimilarity(params.queryVector, 'EmbeddingV1.Data') + 1.0")

	items, err := json.Marshal(tag.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query vector: %w", err)
	}
	script.Params = map[string]json.RawMessage{
		"queryVector": items,
	}

	innerQuery := types.NewQuery()
	innerQuery.MatchAll = types.NewMatchAllQuery()

	query := types.NewQuery()
	query.ScriptScore = types.NewScriptScoreQuery()
	query.ScriptScore.Query = innerQuery
	query.ScriptScore.Script = *script

	result, err := s.client.Search().
		Index(s.index).
		Query(query).
		TrackScores(true).
		Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("tag embedding knn search failed: %w", err)
	}

	hits := result.Hits.Hits
	if len(hits) == 0 {
		return []entity.ElasticTag{}, nil
	}

	// Collect scores sorted ascending to compute percentile cutoff.
	scores := make([]float64, len(hits))
	for i, h := range hits {
		scores[i] = float64(*h.Score_)
	}
	sort.Float64s(scores)
	cutoffIdx := int(float64(len(scores)-1) * float64(percentileMatch))
	cutoffScore := scores[cutoffIdx]

	entities := make([]entity.ElasticTag, 0)
	for _, h := range hits {
		if float64(*h.Score_) < cutoffScore {
			continue
		}
		var t entity.ElasticTag
		if err := json.Unmarshal(h.Source_, &t); err != nil {
			return nil, fmt.Errorf("tag embedding unmarshal failed: %w", err)
		}
		entities = append(entities, t)
	}

	s.slogger.InfoContext(ctx, "SearchTagsByEmbedding",
		"total", len(hits),
		"cutoffScore", cutoffScore,
		"matched", len(entities),
	)
	return entities, nil
}
