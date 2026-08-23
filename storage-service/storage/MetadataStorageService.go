package storage

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
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

//go:embed migrations/metadata
var metadataMigrationFS embed.FS

const imageIdField = "ImageId"
const createdField = "Created"
const scoreField = "_score"

// sort field slices used to build / extract sort keys
var sortFieldsCreated = []string{createdField}
var sortFieldsScore = []string{scoreField, imageIdField}
var sortFieldsImageId = []string{imageIdField}

// extractSortKey builds an ElasticSortKey from the last hit's Sort values,
// using sortFields to name the values in order.
func extractSortKey(resp *search.Response) entity.ElasticSortKey {
	hits := resp.Hits.Hits
	if len(hits) == 0 {
		return nil
	}

	lastSortValues := hits[len(hits)-1].Sort
	if len(lastSortValues) == 0 {
		return nil
	}
	key := make(entity.ElasticSortKey, len(lastSortValues))
	for i := range lastSortValues {
		if i < len(lastSortValues) {
			key[i] = fmt.Sprint(lastSortValues[i])
		}
	}
	return key
}

// buildSortOptions constructs a SortOptions with every field sorted by order.
func buildSortOptions(fields []string, order sortorder.SortOrder) []types.SortCombinations {
	combinations := make([]types.SortCombinations, len(fields))
	for i, field := range fields {
		sortId := types.NewSortOptions()
		fieldSort := *types.NewFieldSort()
		fieldSort.Order = helper.Addr(order)
		sortId.SortOptions[field] = fieldSort
		combinations[i] = sortId
	}
	return combinations
}

// searchAfterValues returns sort values from a sort key in field order,
// ready to pass to SearchAfter(...).
func searchAfterValues(sortKey entity.ElasticSortKey) []types.FieldValue {
	if sortKey == nil {
		return nil
	}
	vals := make([]types.FieldValue, 0, len(sortKey))
	for _, field := range sortKey {
		vals = append(vals, field)
	}
	return vals
}

type MetadataStorageService interface {
	Save(ctx context.Context, file *entity.ElasticImageMetaData) error

	GetAll(
		ctx context.Context,
		sortKey entity.ElasticSortKey,
		pageSize int,
	) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error)

	GetById(ctx context.Context, accountId uuid.UUID, id uuid.UUID) (*entity.ElasticImageMetaData, error)

	GetByAccountIdOrderByCreated(
		ctx context.Context,
		accountId uuid.UUID,
		metadataType *entity.MetadataType,
		sortKey entity.ElasticSortKey,
		pageSize int,
	) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error)

	SearchHybridOrderByScore(
		ctx context.Context,
		accountId uuid.UUID,
		query string,
		embedding entity.EmbeddingItem,
		fuzziness string,
		metadataType *entity.MetadataType,
		sortKey entity.ElasticSortKey,
		pageSize int,
	) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error)

	GetDuplicatesByHash(
		ctx context.Context,
		accountId uuid.UUID,
		hash string,
		excludeIds []uuid.UUID,
		sortKey entity.ElasticSortKey,
		pageSize int,
	) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error)

	GetDuplicatesByEmbeddingOrderByImageId(
		ctx context.Context,
		accountId uuid.UUID,
		embedding entity.EmbeddingItem,
		excludeIds []uuid.UUID,
		threshold float64,
		sortKey entity.ElasticSortKey,
		pageSize int,
	) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error)

	DeleteById(ctx context.Context, accountId uuid.UUID, id uuid.UUID) error
	DeleteByAccountId(ctx context.Context, accountId uuid.UUID) error

	QueryByRaw(ctx context.Context, rawQuery map[string]interface{}, sortKey entity.ElasticSortKey, pageSize int) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error)

	GetRandom(ctx context.Context, accountId uuid.UUID, metadataType *entity.MetadataType) (*entity.ElasticImageMetaData, error)
}

type ElasticMetadataStorageServiceImpl struct {
	*ElasticMigrator

	client    *elasticsearch8.TypedClient
	indexName string
	validate  *validator.Validate
	slogger   *slog.Logger
}

func (e *ElasticMetadataStorageServiceImpl) GetByAccountIdOrderByCreated(
	ctx context.Context,
	accountId uuid.UUID,
	metadataType *entity.MetadataType,
	sortKey entity.ElasticSortKey,
	pageSize int,
) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error) {
	e.slogger.InfoContext(ctx, "ElasticMetadataStorageServiceImpl.searchByAccountId start",
		"sortKey", sortKey,
		"pageSize", pageSize,
	)

	q := types.NewQuery()
	q.Bool = types.NewBoolQuery()
	q.Bool.Filter = []types.Query{*e.accountIdQuery(accountId)}
	if tq := e.typeQuery(metadataType); tq != nil {
		q.Bool.Filter = append(q.Bool.Filter, *tq)
	}

	result, err := e.runQuery(ctx, q, pageSize, sortKey)
	if err != nil {
		return nil, nil, err
	}

	e.slogger.InfoContext(ctx, "ElasticMetadataStorageServiceImpl.searchByAccountId end", "count", len(result.Hits.Hits))

	results, err := e.unmarshalResults(result)
	if err != nil {
		return nil, nil, fmt.Errorf("search_pipeline by account id failed: accountId=%s  sortKey=%v: %w", accountId.String(), sortKey, err)
	}

	return results, extractSortKey(result), nil
}

func (e *ElasticMetadataStorageServiceImpl) runQuery(ctx context.Context, q *types.Query, pageSize int, sortKey entity.ElasticSortKey) (*search.Response, error) {
	searchRequest := e.client.Search().
		Index(e.indexName).
		Query(q).
		Sort(buildSortOptions(sortFieldsCreated, sortorder.Desc)...).
		Size(pageSize)

	if vals := searchAfterValues(sortKey); len(vals) > 0 {
		searchRequest = searchRequest.SearchAfter(vals...)
	}

	result, err := traceElastic(ctx, "search", e.indexName, searchRequest.Do)
	if err != nil {
		return nil, fmt.Errorf("elastic: failed to search_pipeline: response=%s : %w", render.Render(result), err)
	}
	return result, err
}

func (e *ElasticMetadataStorageServiceImpl) GetAll(
	ctx context.Context,
	sortKey entity.ElasticSortKey,
	pageSize int,
) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error) {
	e.slogger.InfoContext(ctx, "ElasticMetadataStorageServiceImpl.GetAll start",
		"sortKey", sortKey,
		"pageSize", pageSize,
	)

	searchRequest := e.client.Search().
		Index(e.indexName).
		Sort(buildSortOptions(sortFieldsImageId, sortorder.Asc)...).
		Size(pageSize)

	if vals := searchAfterValues(sortKey); len(vals) > 0 {
		searchRequest = searchRequest.SearchAfter(vals...)
	}

	result, err := traceElastic(ctx, "search", e.indexName, searchRequest.Do)
	if err != nil {
		return nil, nil, fmt.Errorf("GetAll query failed: response=%s : %w", render.Render(result), err)
	}

	results, err := e.unmarshalResults(result)
	if err != nil {
		return nil, nil, fmt.Errorf("GetAll unmarshall failed: %w", err)
	}

	e.slogger.InfoContext(ctx, "ElasticMetadataStorageServiceImpl.GetAll end", "count", len(results))
	return results, extractSortKey(result), nil
}

func (e *ElasticMetadataStorageServiceImpl) GetDuplicatesByEmbeddingOrderByImageId(
	ctx context.Context,
	accountId uuid.UUID,
	embedding entity.EmbeddingItem,
	excludeIds []uuid.UUID,
	threshold float64,
	sortKey entity.ElasticSortKey,
	pageSize int,
) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error) {
	e.slogger.InfoContext(ctx, "GetDuplicatesByEmbeddingOrderByImageId start", "pageSize", pageSize)

	accountIdFilter := e.accountIdQuery(accountId)
	knnQuery := e.embeddingV1KnnAllQuery(embedding, accountIdFilter, pageSize)
	if len(excludeIds) > 0 {
		excludeQ := types.NewQuery()
		excludeQ.Ids = types.NewIdsQuery()
		excludeQ.Ids.Values = make([]string, len(excludeIds))
		for i, id := range excludeIds {
			excludeQ.Ids.Values[i] = id.String()
		}
		boolExclude := types.NewQuery()
		boolExclude.Bool = types.NewBoolQuery()
		boolExclude.Bool.MustNot = []types.Query{*excludeQ}
		knnQuery.Filter = append(knnQuery.Filter, *boolExclude)
	}

	searchReq := e.client.Search().
		Index(e.indexName).
		Knn(*knnQuery).
		Sort(buildSortOptions(sortFieldsImageId, sortorder.Asc)...).
		TrackScores(true).
		Size(pageSize)

	if vals := searchAfterValues(sortKey); len(vals) > 0 {
		searchReq = searchReq.SearchAfter(vals...)
	}

	resp, err := traceElastic(ctx, "search", e.indexName, searchReq.Do)
	if err != nil {
		return nil, nil, fmt.Errorf("GetDuplicatesByEmbeddingOrderByImageId query failed: %w", err)
	}

	resultsEntity := make([]*entity.ElasticImageMetaData, 0)
	for index, hit := range resp.Hits.Hits {
		if float64(*hit.Score_) < threshold {
			continue
		}
		item, err := unmarshalSearchResultToElasticEntity(index, resp)
		if err != nil {
			return nil, nil, fmt.Errorf("GetDuplicatesByEmbeddingOrderByImageId result unmarshall failed: %w", err)
		}
		if err = e.validate.Struct(item); err != nil {
			return nil, nil, fmt.Errorf("GetDuplicatesByEmbeddingOrderByImageId result validation failed: %w", err)
		}
		resultsEntity = append(resultsEntity, item)
	}

	e.slogger.InfoContext(ctx, "GetDuplicatesByEmbeddingOrderByImageId done", "count", len(resultsEntity))
	return resultsEntity, extractSortKey(resp), nil
}

func (e *ElasticMetadataStorageServiceImpl) DeleteByAccountId(ctx context.Context, accountId uuid.UUID) error {
	e.slogger.InfoContext(ctx, "DeleteByAccountId: delete request", "accountId", accountId)

	deleteRequest := e.client.DeleteByQuery(e.indexName).
		Refresh(true).
		Query(e.accountIdQuery(accountId))
	result, err := traceElastic(ctx, "delete_by_query", e.indexName, deleteRequest.Do)

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

	deleteRequest := e.client.DeleteByQuery(e.indexName).
		Refresh(true).
		Query(query)
	result, err := traceElastic(ctx, "delete_by_query", e.indexName, deleteRequest.Do)

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

	searchRequest := e.client.Search().
		Index(e.indexName).
		Query(query)
	result, err := traceElastic(ctx, "search", e.indexName, searchRequest.Do)

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

func (e *ElasticMetadataStorageServiceImpl) GetDuplicatesByHash(
	ctx context.Context,
	accountId uuid.UUID,
	hash string,
	excludeIds []uuid.UUID,
	sortKey entity.ElasticSortKey,
	pageSize int,
) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error) {
	e.slogger.InfoContext(ctx, "GetDuplicatesByHash: call",
		"hash", hash)

	query := types.NewQuery()

	query.Bool = types.NewBoolQuery()
	query.Bool.Must = []types.Query{
		*e.accountIdQuery(accountId),
		*e.hashQuery(hash),
	}
	if len(excludeIds) > 0 {
		excludeQ := types.NewQuery()
		excludeQ.Ids = types.NewIdsQuery()
		excludeQ.Ids.Values = make([]string, len(excludeIds))
		for i, id := range excludeIds {
			excludeQ.Ids.Values[i] = id.String()
		}
		query.Bool.MustNot = []types.Query{*excludeQ}
	}

	searchRequest := e.client.Search().
		Index(e.indexName).
		Query(query).
		Sort(buildSortOptions(sortFieldsImageId, sortorder.Desc)...).
		Size(pageSize)

	if vals := searchAfterValues(sortKey); len(vals) > 0 {
		searchRequest = searchRequest.SearchAfter(vals...)
	}

	result, err := traceElastic(ctx, "search", e.indexName, searchRequest.Do)
	if err != nil {
		return nil, nil, fmt.Errorf("elastic: failed to search_pipeline: response=%s : %w", render.Render(result), err)
	}

	resultsSize := len(result.Hits.Hits)
	if resultsSize == 0 {
		return []*entity.ElasticImageMetaData{}, nil, nil
	}

	data := make([]*entity.ElasticImageMetaData, resultsSize)
	for i := range resultsSize {
		item, err := unmarshalSearchResultToElasticEntity(i, result)
		if err != nil {
			return nil, nil, fmt.Errorf("GetDuplicatesByHash result unmarshall failed: id: %s error: %w", hash, err)
		}
		data[i] = item
	}

	return data, extractSortKey(result), nil
}

func (e *ElasticMetadataStorageServiceImpl) SearchHybridOrderByScore(
	ctx context.Context,
	accountId uuid.UUID,
	query string,
	embedding entity.EmbeddingItem,
	fuzziness string,
	metadataType *entity.MetadataType,
	sortKey entity.ElasticSortKey,
	pageSize int,
) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error) {
	e.slogger.InfoContext(ctx, "SearchHybridOrderByScore start", "query", query, "pageSize", pageSize)

	accountIdFilter := e.accountIdQuery(accountId)
	bm25Query := e.stringAndAccountQuery(accountId, query, fuzziness)
	if tq := e.typeQuery(metadataType); tq != nil {
		bm25Query.Bool.Must = append(bm25Query.Bool.Must, *tq)
	}
	knnFilters := []types.Query{*accountIdFilter}
	if tq := e.typeQuery(metadataType); tq != nil {
		knnFilters = append(knnFilters, *tq)
	}
	knnQuery := e.embeddingV1KnnAllQueryWithFilters(embedding, knnFilters, pageSize)

	searchReq := e.client.Search().
		Index(e.indexName).
		Query(bm25Query).
		Knn(*knnQuery).
		Sort(buildSortOptions(sortFieldsScore, sortorder.Desc)...).
		TrackScores(true).
		Size(pageSize)

	if vals := searchAfterValues(sortKey); len(vals) > 0 {
		searchReq = searchReq.SearchAfter(vals...)
	}

	resp, err := traceElastic(ctx, "search", e.indexName, searchReq.Do)
	if err != nil {
		return nil, nil, fmt.Errorf("SearchHybridOrderByScore query failed: %w", err)
	}

	resultsEntity := make([]*entity.ElasticImageMetaData, 0)
	for index := range resp.Hits.Hits {
		item, err := unmarshalSearchResultToElasticEntity(index, resp)
		if err != nil {
			return nil, nil, fmt.Errorf("SearchHybridOrderByScore result unmarshall failed: %w", err)
		}
		if err = e.validate.Struct(item); err != nil {
			return nil, nil, fmt.Errorf("SearchHybridOrderByScore result validation failed: %w", err)
		}
		resultsEntity = append(resultsEntity, item)
	}

	e.slogger.InfoContext(ctx, "SearchHybridOrderByScore done", "count", len(resultsEntity))
	return resultsEntity, extractSortKey(resp), nil
}

func (e *ElasticMetadataStorageServiceImpl) Save(ctx context.Context, file *entity.ElasticImageMetaData) error {

	file.Updated = time.Now().UnixMicro()

	buff := bytes.NewBuffer(nil)
	jsonEncoder := json.NewEncoder(buff)
	err := jsonEncoder.Encode(file)
	if err != nil {
		return fmt.Errorf("json encode failed: %w", err)
	}

	indexRequest := e.client.
		Index(e.indexName).
		Document(file).
		Id(file.ImageId.String()).
		Refresh(refresh.True)
	response, err := traceElastic(ctx, "index", e.indexName, indexRequest.Do)

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

func (e *ElasticMetadataStorageServiceImpl) QueryByRaw(
	ctx context.Context,
	rawQuery map[string]interface{},
	sortKey entity.ElasticSortKey,
	pageSize int,
) ([]*entity.ElasticImageMetaData, entity.ElasticSortKey, error) {
	e.slogger.InfoContext(ctx, "QueryByRaw start", "pageSize", pageSize)

	body := map[string]interface{}{
		"query": rawQuery,
		"sort":  []map[string]interface{}{{imageIdField: map[string]interface{}{"order": "asc"}}},
		"size":  pageSize,
	}
	if sortKey != nil {
		body["search_after"] = []interface{}{sortKey[0]}
	}

	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return nil, nil, fmt.Errorf("QueryByRaw: marshal body failed: %w", err)
	}

	searchRequest := e.client.Search().Index(e.indexName).Raw(bytes.NewReader(jsonBytes))
	result, err := traceElastic(ctx, "search_raw", e.indexName, searchRequest.Do)
	if err != nil {
		return nil, nil, fmt.Errorf("QueryByRaw: search failed: %w", err)
	}

	results, err := e.unmarshalResults(result)
	if err != nil {
		return nil, nil, fmt.Errorf("QueryByRaw: unmarshal failed: %w", err)
	}

	e.slogger.InfoContext(ctx, "QueryByRaw done", "count", len(results))
	return results, extractSortKey(result), nil
}

func (e *ElasticMetadataStorageServiceImpl) embeddingV1KnnAllQuery(
	img entity.EmbeddingItem,
	accountIdQuery *types.Query,
	count int,
) *types.KnnSearch {
	return e.embeddingV1KnnAllQueryWithFilters(img, []types.Query{*accountIdQuery}, count)
}

func (e *ElasticMetadataStorageServiceImpl) embeddingV1KnnAllQueryWithFilters(
	img entity.EmbeddingItem,
	filters []types.Query,
	count int,
) *types.KnnSearch {
	query := types.NewKnnSearch()
	query.Field = "EmbeddingList.Data"
	query.QueryVector = img.Data
	query.NumCandidates = helper.Addr(1000)
	query.K = helper.Addr(count)
	query.Filter = filters
	return query
}

func (e *ElasticMetadataStorageServiceImpl) typeQuery(t *entity.MetadataType) *types.Query {
	if t == nil {
		return nil
	}
	q := types.NewQuery()
	q.Match = map[string]types.MatchQuery{
		"Type": {Query: string(*t), Fuzziness: 0, Operator: &operator.And},
	}
	return q
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
	query.Match = map[string]types.MatchQuery{
		"Hash": {
			Query:     hash,
			Fuzziness: 0,
			Operator:  &operator.And,
		},
	}
	return query
}

func (e *ElasticMetadataStorageServiceImpl) stringAndAccountQuery(
	accountId uuid.UUID,
	queryString string,
	fuzziness string,
) *types.Query {
	q1 := types.NewQuery()
	q1.Match = map[string]types.MatchQuery{
		"Result": {
			Query:     queryString,
			Fuzziness: fuzziness,
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

func (e *ElasticMetadataStorageServiceImpl) GetRandom(
	ctx context.Context,
	accountId uuid.UUID,
	metadataType *entity.MetadataType,
) (*entity.ElasticImageMetaData, error) {
	query := types.NewQuery()
	query.Bool = types.NewBoolQuery()
	query.Bool.Filter = []types.Query{*e.accountIdQuery(accountId)}
	if metadataType != nil {
		typeQ := types.NewQuery()
		typeQ.Match = map[string]types.MatchQuery{
			"Type": {Query: string(*metadataType), Fuzziness: 0, Operator: &operator.And},
		}
		query.Bool.Filter = append(query.Bool.Filter, *typeQ)
	}

	countRequest := e.client.Search().
		Index(e.indexName).
		Query(query).
		Size(0)
	countResp, err := traceElastic(ctx, "search_count", e.indexName, countRequest.Do)
	if err != nil {
		return nil, fmt.Errorf("GetRandom: count failed: %w", err)
	}
	total := countResp.Hits.Total.Value
	if total == 0 {
		return nil, nil
	}

	offset := rand.Intn(int(total))
	searchRequest := e.client.Search().
		Index(e.indexName).
		Query(query).
		From(offset).
		Size(1)
	resp, err := traceElastic(ctx, "search", e.indexName, searchRequest.Do)
	if err != nil {
		return nil, fmt.Errorf("GetRandom: search failed: %w", err)
	}

	results, err := e.unmarshalResults(resp)
	if err != nil {
		return nil, fmt.Errorf("GetRandom: unmarshal failed: %w", err)
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results[0], nil
}

func NewElasticMetadataStorage(
	cfg *conf.Config,
	validate *validator.Validate,
) (MetadataStorageService, error) {
	config := cfg.MetadataDb
	configEmbeddings := cfg.Extracting
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

	migrator, err := NewElasticMigrator(
		config.Elastic,
		metadataMigrationFS, "migrations/metadata",
		config.Index, MigrationHistoryIndex,
		map[string]string{
			"index": config.Index,
			"dims":  strconv.Itoa(configEmbeddings.EmbeddingDimensions),
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("create metadata migrator failed: %w", err)
	}

	return &ElasticMetadataStorageServiceImpl{
		ElasticMigrator: migrator,
		client:          es8,
		indexName:       config.Index,
		validate:        validate,
		slogger:         logger,
	}, nil
}

func NewMetadataStorageService(cfg *conf.Config, validate *validator.Validate) (MetadataStorageService, error) {
	return NewElasticMetadataStorage(cfg, validate)
}
