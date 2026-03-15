package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/operator"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/refresh"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
	"github.com/gdexlab/go-render/render"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/entity"
)

const MaxFuzzy = 10

type MetadataStorageService interface {
	Save(ctx context.Context, file *entity.ElasticImageMetaData) error

	List(
		ctx context.Context,
		accountId *uuid.UUID,
		id *uuid.UUID,
		idAfter *uuid.UUID,
		pageSize *int,
	) ([]*entity.ElasticImageMetaData, error)

	SearchByAccountId(ctx context.Context,
		accountId uuid.UUID,
		idAfter *uuid.UUID,
		pageSize *int,
	) ([]*entity.ElasticImageMetaData, error)

	SearchSimple(ctx context.Context,
		accountId uuid.UUID,
		query string,
		idAfter *uuid.UUID,
		pageSize *int,
	) ([]*entity.ElasticImageMetaData, error)

	SearchFuzzy(ctx context.Context,
		accountId uuid.UUID,
		query string,
		pageSize *int,
	) ([]*entity.ElasticImageMetaData, error)

	GetById(ctx context.Context, accountId uuid.UUID, id uuid.UUID) (*entity.ElasticImageMetaData, error)
	GetByHash(ctx context.Context, accountId uuid.UUID, hash string, count *int) ([]*entity.ElasticImageMetaData, error)
	SearchByEmbeddingV1(ctx context.Context, accountId uuid.UUID, img entity.ElasticEmbeddingV1, count int, filterSimilarity bool) ([]*entity.ElasticImageMetaData, error)

	DeleteById(ctx context.Context, accountId uuid.UUID, id uuid.UUID) error
	DeleteByAccountId(ctx context.Context, accountId uuid.UUID) error
}

type ElasticMetadataStorageServiceImpl struct {
	client                 *elasticsearch8.TypedClient
	embeddingMatchTreshold float64
	indexName              string
	validate               *validator.Validate
	slogger                *slog.Logger
}

func (e *ElasticMetadataStorageServiceImpl) SearchByAccountId(
	ctx context.Context,
	accountId uuid.UUID,
	sortIdAfter *uuid.UUID,
	pageSize *int,
) ([]*entity.ElasticImageMetaData, error) {
	result, err := e.searchByAccountId(ctx, accountId, sortIdAfter, pageSize)
	if err != nil {
		return nil, fmt.Errorf("search_pipeline by account id failed: accountId=%s  sortIdAfter=%v: %w", accountId.String(), sortIdAfter, err)
	}

	results, err := e.unmarshalResults(result)
	if err != nil {
		return nil, fmt.Errorf("search_pipeline by account id failed: accountId=%s  sortIdAfter=%v: %w", accountId.String(), sortIdAfter, err)
	}

	return results, nil
}
func (e *ElasticMetadataStorageServiceImpl) List(
	ctx context.Context,
	accountId *uuid.UUID,
	id *uuid.UUID,
	sortIdAfter *uuid.UUID,
	pageSize *int,
) ([]*entity.ElasticImageMetaData, error) {
	result, err := e.list(ctx, accountId, id, sortIdAfter, pageSize)
	if err != nil {
		return nil, fmt.Errorf("list all failed: sortIdAfter=%v: %w", sortIdAfter, err)
	}

	results, err := e.unmarshalResults(result)
	if err != nil {
		return nil, fmt.Errorf("list unmarshall failed: sortIdAfter=%v: %w", sortIdAfter, err)
	}

	return results, nil
}

func (e *ElasticMetadataStorageServiceImpl) SearchSimple(ctx context.Context, accountId uuid.UUID, query string, sortIdAfter *uuid.UUID, pageSize *int) ([]*entity.ElasticImageMetaData, error) {
	result, err := e.searchSimple(ctx, accountId, query, sortIdAfter, pageSize)
	if err != nil {
		return nil, fmt.Errorf("search_pipeline all failed: %w", err)
	}

	results, err := e.unmarshalResults(result)
	if err != nil {
		return nil, fmt.Errorf("result unmarshall failed: %w", err)
	}

	return results, nil
}

func (e *ElasticMetadataStorageServiceImpl) SearchFuzzy(ctx context.Context, accountId uuid.UUID, query string, pageSize *int) ([]*entity.ElasticImageMetaData, error) {
	result, err := e.searchFuzzy(ctx, accountId, query, pageSize)
	if err != nil {
		return nil, fmt.Errorf("search_pipeline all failed: %w", err)
	}

	results, err := e.unmarshalResults(result)
	if err != nil {
		return nil, fmt.Errorf("result unmarshall failed: %w", err)
	}

	return results, nil
}

func (e *ElasticMetadataStorageServiceImpl) DeleteByAccountId(ctx context.Context, accountId uuid.UUID) error {
	e.slogger.InfoContext(ctx, "DeleteByAccountId: delete request", "accountId", accountId)

	result, err := e.client.DeleteByQuery(e.indexName).
		Refresh(true).
		Query(e.accountIdQuery(accountId)).
		Do(ctx)

	if err != nil {
		return fmt.Errorf("DeleteByAccountId query failed: %w", err)
	}

	e.slogger.InfoContext(ctx, "DeleteByAccountId: delete response", "accountId", accountId, "deleted", result.Deleted)
	return nil
}

func (e *ElasticMetadataStorageServiceImpl) DeleteById(ctx context.Context, accountId uuid.UUID, id uuid.UUID) error {
	e.slogger.InfoContext(ctx, "DeleteById: delete request",
		"id", id)

	query := types.NewQuery()
	query.Bool = types.NewBoolQuery()
	query.Bool.Must = []types.Query{
		*e.idQuery(id),
		*e.accountIdQuery(accountId),
	}

	result, err := e.client.DeleteByQuery(e.indexName).
		Refresh(true).
		Query(query).
		Do(ctx)

	if err != nil {
		return fmt.Errorf("DeleteById query falied: %w", err)
	}

	e.slogger.InfoContext(ctx, "DeleteById: delete response", "id", id)
	e.slogger.DebugContext(ctx, "DeleteById: delete response details", "id", id, "result", result)
	return nil
}

func (e *ElasticMetadataStorageServiceImpl) GetById(ctx context.Context, accountId uuid.UUID, id uuid.UUID) (*entity.ElasticImageMetaData, error) {
	e.slogger.InfoContext(ctx, "GetById: call",
		"id", id.String())

	query := types.NewQuery()

	query.Bool = types.NewBoolQuery()
	query.Bool.Must = []types.Query{
		*e.accountIdQuery(accountId),
		*e.idQuery(id),
	}

	result, err := e.client.Search().
		Index(e.indexName).
		Query(query).
		Do(ctx)

	if err != nil {
		return nil, err
	}

	if len(result.Hits.Hits) == 0 {
		return nil, nil
	}

	data, err := unmarshalSearchResultToElasticEntity(0, result)
	if err != nil {
		return nil, fmt.Errorf("GetById result unmarshall failed: id: %s error: %w", id.String(), err)
	}

	return data, e.validate.Struct(data)
}

func (e *ElasticMetadataStorageServiceImpl) GetByHash(
	ctx context.Context,
	accountId uuid.UUID,
	hash string,
	count *int,
) ([]*entity.ElasticImageMetaData, error) {
	e.slogger.InfoContext(ctx, "GetByHash: call",
		"hash", hash)

	query := types.NewQuery()

	query.Bool = types.NewBoolQuery()
	query.Bool.Must = []types.Query{
		*e.accountIdQuery(accountId),
		*e.hashQuery(hash),
	}

	result, err := e.runSearchQuery(ctx, query, nil, count)

	if err != nil {
		return nil, fmt.Errorf("runSearchQuery falied: %w", err)
	}

	resultsSize := len(result.Hits.Hits)
	if resultsSize == 0 {
		return []*entity.ElasticImageMetaData{}, nil
	}

	data := make([]*entity.ElasticImageMetaData, resultsSize)
	for i := range resultsSize {
		item, err := unmarshalSearchResultToElasticEntity(i, result)
		if err != nil {
			return nil, fmt.Errorf("GetByHash result unmarshall failed: id: %s error: %w", hash, err)
		}
		data[i] = item
	}

	return data, nil
}

func (e *ElasticMetadataStorageServiceImpl) SearchByEmbeddingV1(
	ctx context.Context,
	accountId uuid.UUID,
	img entity.ElasticEmbeddingV1,
	count int,
	filterSimilarity bool,
) ([]*entity.ElasticImageMetaData, error) {
	e.slogger.InfoContext(ctx, "Search embedding start",
		"pageSize", count,
	)

	accountIdQuery := e.accountIdQuery(accountId)
	knnQuery := e.embeddingV1KnnAllQuery(img, accountIdQuery, count)
	result, err := e.processKnn(ctx, *knnQuery)
	if err != nil {
		return nil, fmt.Errorf("knn search_pipeline failed: %w", err)
	}

	resultsSize := len(result.Hits.Hits)

	resultsEntity := make([]*entity.ElasticImageMetaData, 0)
	for index := range resultsSize {
		if filterSimilarity && float64(*(result.Hits.Hits[index].Score_)) < e.embeddingMatchTreshold {
			continue
		}

		item, err := unmarshalSearchResultToElasticEntity(index, result)
		if err != nil {
			return nil, fmt.Errorf("SearchByEmbeddingV1 result unmarshall failed: error: %w", err)
		}

		err = e.validate.Struct(item)
		if err != nil {
			return nil, fmt.Errorf("SearchByEmbeddingV1 result vaildation failed: error: %w", err)
		}

		resultsEntity = append(resultsEntity, item)
	}

	e.slogger.InfoContext(ctx, "Search embedding result",
		"count", len(resultsEntity),
	)
	return resultsEntity, nil
}

func (e *ElasticMetadataStorageServiceImpl) Save(ctx context.Context, file *entity.ElasticImageMetaData) error {

	file.Updated = time.Now().UnixMicro()

	buff := bytes.NewBuffer(nil)
	jsonEncoder := json.NewEncoder(buff)
	err := jsonEncoder.Encode(file)
	if err != nil {
		return fmt.Errorf("json encode failed: %w", err)
	}

	response, err := e.client.
		Index(e.indexName).
		Document(file).
		Id(file.ImageId.String()).
		Refresh(refresh.True).
		Do(ctx)

	if err != nil {
		return fmt.Errorf("save metadata document error: id=%s : %w", file.ImageId, err)
	}

	e.slogger.InfoContext(ctx, "Save metadata document",
		"id", file.ImageId)

	e.slogger.DebugContext(ctx, "Save metadata document details",
		"id", file.ImageId,
		"response", render.Render(response))

	return err
}

func (e *ElasticMetadataStorageServiceImpl) embeddingV1KnnAllQuery(
	img entity.ElasticEmbeddingV1,
	accountIdQuery *types.Query,
	count int,
) *types.KnnSearch {

	query := types.NewKnnSearch()
	query.Field = "EmbeddingV1.Data"
	query.QueryVector = *img.Data
	query.NumCandidates = helper.Addr(1000)
	query.K = helper.Addr(count)
	query.Filter = []types.Query{*accountIdQuery}
	return query
}

func (e *ElasticMetadataStorageServiceImpl) accountIdQuery(accountId uuid.UUID) *types.Query {
	accountIdQuery := types.NewQuery()
	accountIdQuery.Match = map[string]types.MatchQuery{
		"AccountId": {
			Query:     accountId.String(),
			Fuzziness: 0,
			Operator:  &operator.And,
		},
	}
	return accountIdQuery
}

func (e *ElasticMetadataStorageServiceImpl) idQuery(
	id uuid.UUID,
) *types.Query {
	q1 := types.NewQuery()
	q1.Ids = types.NewIdsQuery()
	q1.Ids.Values = []string{id.String()}
	return q1
}
func (e *ElasticMetadataStorageServiceImpl) hashQuery(
	hash string,
) *types.Query {
	query := types.NewQuery()
	query.QueryString = types.NewQueryStringQuery()
	query.QueryString.Query = fmt.Sprintf("CalcHash: \"%s\"", hash)
	return query
}

func (e *ElasticMetadataStorageServiceImpl) stringAndAccountQuery(
	accountId uuid.UUID,
	queryString string,
) *types.Query {
	q1 := types.NewQuery()
	q1.Match = map[string]types.MatchQuery{
		"Result": {
			Query:     queryString,
			Fuzziness: "0",
			Operator:  &operator.And,
		},
	}

	query := types.NewQuery()
	query.Bool = types.NewBoolQuery()
	query.Bool.Must = []types.Query{
		*q1, *e.accountIdQuery(accountId),
	}
	return query
}

func (e *ElasticMetadataStorageServiceImpl) fuzzyStringAndAccountQuery(
	accountId uuid.UUID,
	queryString string,
) *types.Query {
	q1 := types.NewQuery()
	q1.Match = map[string]types.MatchQuery{
		"Result": {
			Query:     queryString,
			Fuzziness: "AUTO",
			Operator:  &operator.And,
		},
	}

	query := types.NewQuery()
	query.Bool = types.NewBoolQuery()
	query.Bool.Must = []types.Query{
		*q1, *e.accountIdQuery(accountId),
	}
	return query
}

// Search implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) processKnn(
	ctx context.Context,
	knnQuery types.KnnSearch,
) (*search.Response, error) {

	sortId := types.NewSortOptions()
	sortConfig := types.NewFieldSort()
	sortConfig.Order = &sortorder.Desc

	sortId.SortOptions["_score"] = *sortConfig

	searchRequest := e.client.Search().
		Index(e.indexName).
		Knn(knnQuery).
		Sort(sortId).
		TrackScores(true)

	knn, err := searchRequest.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("elastic: failed to knn-search_pipeline: response=%s : %w", render.Render(knn), err)
	}

	return knn, nil
}

// Search implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) runSearchQuery(
	ctx context.Context,
	query *types.Query,
	idAfter *uuid.UUID,
	size *int,
) (*search.Response, error) {
	highlight := types.NewHighlight()
	highlight.PreTags = []string{"<MATCH>"}
	highlight.PostTags = []string{"</MATCH>"}
	highlight.Fields = map[string]types.HighlightField{
		"Result": *types.NewHighlightField(),
	}

	resultField := types.NewFieldAndFormat()
	resultField.Field = "Result"

	sortId := types.NewSortOptions()
	sortId.SortOptions["ImageId"] = *types.NewFieldSort()

	searchRequest := e.client.Search().
		Index(e.indexName).
		Query(query).
		Fields(*resultField).
		Highlight(highlight).
		Sort(sortId)

	if idAfter != nil {
		searchRequest = searchRequest.SearchAfter(*idAfter)
	}

	if size != nil {
		searchRequest = searchRequest.Size(*size)
	}

	resp, err := searchRequest.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("elastic: failed to search_pipeline: response=%s : %w", render.Render(resp), err)
	}

	return resp, nil
}

func (e *ElasticMetadataStorageServiceImpl) searchFuzzy(ctx context.Context, accountId uuid.UUID, queryString string, pageSize *int) (*search.Response, error) {
	e.slogger.InfoContext(ctx, "Search FUZZY start",
		"query", queryString,
		"pageSize", pageSize,
	)

	queryFuzzy := e.fuzzyStringAndAccountQuery(accountId, queryString)
	resultFuzzy, err := e.runSearchQuery(ctx, queryFuzzy, nil, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to search_pipeline FUZZY query : %w", err)
	}

	e.slogger.InfoContext(ctx, "Search FUZZY result", "count", len(resultFuzzy.Hits.Hits))

	return resultFuzzy, nil
}

func (e *ElasticMetadataStorageServiceImpl) searchSimple(ctx context.Context, accountId uuid.UUID, queryString string, idAfter *uuid.UUID, pageSize *int) (*search.Response, error) {
	e.slogger.InfoContext(ctx, "Search SIMPLE start",
		"idAfter", idAfter,
		"query", queryString,
		"pageSize", pageSize,
	)

	querySimple := e.stringAndAccountQuery(accountId, queryString)
	resultSimple, err := e.runSearchQuery(ctx, querySimple, idAfter, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to search_pipeline SIMPLE query : %w", err)
	}

	e.slogger.InfoContext(ctx, "Search SIMPLE result", "count", len(resultSimple.Hits.Hits))

	return resultSimple, nil
}

func (e *ElasticMetadataStorageServiceImpl) list(
	ctx context.Context,
	accountId *uuid.UUID,
	id *uuid.UUID,
	idAfter *uuid.UUID,
	pageSize *int) (*search.Response, error) {
	e.slogger.InfoContext(ctx, "ElasticMetadataStorageServiceImpl.list start",
		"idAfter", idAfter,
		"pageSize", pageSize,
	)
	query := types.NewQuery()
	query.Bool = types.NewBoolQuery()

	queries := make([]types.Query, 0)
	if accountId != nil {
		queries = append(queries, *e.accountIdQuery(*accountId))
	}

	if id != nil {
		queries = append(queries, *e.idQuery(*id))
	}

	query.Bool.Must = queries

	result, err := e.runSearchQuery(ctx, query, idAfter, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to search_pipeline all query : %w", err)
	}

	e.slogger.InfoContext(ctx, "ElasticMetadataStorageServiceImpl.list end", "count", len(result.Hits.Hits))
	return result, nil
}

func (e *ElasticMetadataStorageServiceImpl) searchByAccountId(ctx context.Context, accountId uuid.UUID, idAfter *uuid.UUID, pageSize *int) (*search.Response, error) {
	e.slogger.InfoContext(ctx, "ElasticMetadataStorageServiceImpl.searchByAccountId start",
		"idAfter", idAfter,
		"pageSize", pageSize,
	)

	q := e.accountIdQuery(accountId)
	result, err := e.runSearchQuery(ctx, q, idAfter, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to search_pipeline all query : %w", err)
	}

	e.slogger.InfoContext(ctx, "ElasticMetadataStorageServiceImpl.searchByAccountId end", "count", len(result.Hits.Hits))
	return result, nil
}

func (e *ElasticMetadataStorageServiceImpl) unmarshalResults(result *search.Response) ([]*entity.ElasticImageMetaData, error) {
	resultsSize := len(result.Hits.Hits)
	results := make([]*entity.ElasticImageMetaData, resultsSize)

	for index := range resultsSize {
		item, err := unmarshalSearchResultToElasticEntity(index, result)
		if err != nil {
			return nil, fmt.Errorf("unmarshall failed: %w", err)
		}
		err = e.validate.Struct(item)
		if err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
		results[index] = item
	}
	return results, nil
}

func unmarshalSearchResultToElasticEntity(i int, result *search.Response) (*entity.ElasticImageMetaData, error) {
	hits := result.Hits.Hits
	if len(hits) == 0 {
		return nil, nil
	}

	hit := hits[i]

	return unmarshalSourceDocument(hit.Source_)
}

func unmarshalSourceDocument(result json.RawMessage) (*entity.ElasticImageMetaData, error) {

	var document entity.ElasticImageMetaData
	err := json.Unmarshal(result, &document)
	return &document, err
}

func NewElasticMetadataStorage(
	config *conf.MetadataStorageConfig,
	validate *validator.Validate,
) (MetadataStorageService, error) {
	es8, _ := elasticsearch8.NewTypedClient(*config.Elastic)
	logger := slog.With("service", "ElasticMetadataStorage")
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
	indexTypeMapping.Properties["AccountId"] = types.NewKeywordProperty()
	indexTypeMapping.Properties["CalcHash"] = types.NewKeywordProperty()
	indexTypeMapping.Properties["ImageId"] = types.NewKeywordProperty()
	indexTypeMapping.Properties["Tags"] = types.NewKeywordProperty()

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

	return &ElasticMetadataStorageServiceImpl{
		client:                 es8,
		embeddingMatchTreshold: config.EmbeddingMatchTreshold,
		indexName:              config.Index,
		validate:               validate,
		slogger:                logger,
	}, nil
}

func NewMetadataStorageService(config *conf.MetadataStorageConfig, validate *validator.Validate) (MetadataStorageService, error) {
	return NewElasticMetadataStorage(config, validate)
}
