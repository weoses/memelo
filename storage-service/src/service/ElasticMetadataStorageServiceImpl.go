package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/operator"
	"github.com/gdexlab/go-render/render"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"mine.local/ocr-gallery/common/commonconst"
	"mine.local/ocr-gallery/storage-service/conf"
	"mine.local/ocr-gallery/storage-service/entity"
)

const INDEX_NAME = "image-metadata"
const MAX_FUZZY = 10

type ElasticMetadataStorageServiceImpl struct {
	client                 *elasticsearch8.TypedClient
	embeddingMatchTreshold float64
	validate               *validator.Validate
}

func (e *ElasticMetadataStorageServiceImpl) embeddingV1KnnQuery(
	img *entity.ElasticEmbeddingV1,
	count int,
) *types.KnnSearch {

	query := types.NewKnnSearch()
	query.Field = "EmbeddingV1.Data"
	query.QueryVector = *img.Data
	query.NumCandidates = addr(1000)
	query.K = addr(count)
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

func (e *ElasticMetadataStorageServiceImpl) allQuery(accountId uuid.UUID) *types.Query {
	query := types.NewQuery()
	query.Bool = types.NewBoolQuery()
	query.Bool.Must = []types.Query{
		*e.accountIdQuery(accountId),
	}
	return query
}

func (e *ElasticMetadataStorageServiceImpl) simpleQuery(
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

func (e *ElasticMetadataStorageServiceImpl) fuzzySearchQuery(
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

// Delete implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) Delete(ctx context.Context, id uuid.UUID) error {
	response, err := e.client.
		Delete(INDEX_NAME, id.String()).
		Do(ctx)
	if err != nil {
		return fmt.Errorf("elastic failed to delete: %w", err)
	}

	slog.Info("Delete metadata document ",
		"id", id,
		"response", render.Render(response))

	return err
}

// Search implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) processKnn(
	ctx context.Context,
	query types.KnnSearch,
) (*search.Response, error) {

	sortId := types.NewSortOptions()
	sortId.SortOptions["Created"] = *types.NewFieldSort()

	search := e.client.Search().
		Index(INDEX_NAME).
		Sort(sortId).
		Knn(query).
		TrackScores(true)

	knn, err := search.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("elastic: failed to knn-search: response=%s : %w", render.Render(knn), err)
	}

	return knn, nil
}

// Search implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) runSearchQuery(
	ctx context.Context,
	query *types.Query,
	sortIdAfter *int64,
	pageSize *int,
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
	sortId.SortOptions["Created"] = *types.NewFieldSort()

	search := e.client.Search().
		Index(INDEX_NAME).
		Query(query).
		Fields(*resultField).
		Highlight(highlight).
		Sort(sortId)

	if sortIdAfter != nil {
		search = search.SearchAfter(*sortIdAfter)
	}

	if pageSize != nil {
		search = search.Size(*pageSize)
	}

	resp, err := search.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("elastic: failed to search: response=%s : %w", render.Render(resp), err)
	}

	return resp, nil
}

func (e *ElasticMetadataStorageServiceImpl) searchQueryInternal(
	ctx context.Context,
	accountId uuid.UUID,
	queryString string,
	sortIdAfter *int64,
	pageSize *int,
) (*search.Response, error) {
	if pageSize != nil && *pageSize <= 0 {
		return nil, errors.New("page size is zero")
	}

	if queryString == "" {
		slog.Info("Search ALL (no query string)",
			commonconst.ACCOUNTID_LOG, accountId,
			commonconst.OFFSET_LOG, sortIdAfter,
			"pageSize", pageSize,
		)

		q := e.allQuery(accountId)
		result, err := e.runSearchQuery(ctx, q, sortIdAfter, pageSize)
		if err != nil {
			return nil, fmt.Errorf("failed to search all query : %w", err)
		}

		slog.Info("Search ALL result", "count", len(result.Hits.Hits))
		return result, nil
	}

	slog.Info("Search SIMPLE",
		commonconst.ACCOUNTID_LOG, accountId,
		commonconst.OFFSET_LOG, sortIdAfter,
		commonconst.QUERY_LOG, queryString,
		"pageSize", pageSize,
	)
	q := e.simpleQuery(accountId, queryString)
	result, err := e.runSearchQuery(ctx, q, sortIdAfter, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to search SIMPLE query : %w", err)
	}

	slog.Info("Search SIMPLE result", "count", len(result.Hits.Hits))

	resultsSize := len(result.Hits.Hits)
	if resultsSize > 0 || sortIdAfter != nil {
		return result, nil
	}

	slog.Info("Search FUZZY",
		commonconst.ACCOUNTID_LOG, accountId,
		commonconst.OFFSET_LOG, sortIdAfter,
		commonconst.QUERY_LOG, queryString,
		"maxCount", MAX_FUZZY,
	)

	q = e.fuzzySearchQuery(accountId, queryString)
	result, err = e.runSearchQuery(ctx, q, sortIdAfter, addr(MAX_FUZZY))
	if err != nil {
		return nil, fmt.Errorf("failed to search FUZZY query : %w", err)
	}

	return result, nil
}

// Search implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) Search(
	ctx context.Context,
	accountId uuid.UUID,
	queryString string,
	sortIdAfter *int64,
	pageSize *int,
) ([]*entity.ElasticMatchedContent, error) {

	result, err := e.searchQueryInternal(ctx, accountId, queryString, sortIdAfter, pageSize)
	if err != nil {
		return nil,
			fmt.Errorf(
				"search failed: accountId: %s queryString: %s sortIdAfter: %v error: %w",
				accountId.String(), queryString, sortIdAfter, err)
	}

	resultsSize := len(result.Hits.Hits)
	results := make([]*entity.ElasticMatchedContent, resultsSize)

	for index := range resultsSize {
		item, err := unmarhalSearchResultToMatchedContent(index, result)
		if err != nil {
			return nil,
				fmt.Errorf(
					"search result unmarshall failed: accountId: %s queryString: %s sortIdAfter: %v error: %w",
					accountId.String(), queryString, sortIdAfter, err)
		}
		err = e.validate.Struct(item)
		if err != nil {
			return nil,
				fmt.Errorf(
					"search result vaildation failed: accountId: %s queryString: %s sortIdAfter: %v error: %w",
					accountId.String(), queryString, sortIdAfter, err)
		}
		results[index] = item
	}
	return results, nil
}

// GetById implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) GetById(ctx context.Context, id uuid.UUID) (*entity.ElasticImageMetaData, error) {
	slog.Info("GetById: call",
		"id", id.String())

	query := types.NewQuery()
	query.Ids = types.NewIdsQuery()
	query.Ids.Values = []string{id.String()}

	result, err := e.client.Search().
		Index(INDEX_NAME).
		Query(query).
		Do(ctx)

	if err != nil {
		return nil, err
	}

	data, err := unmarhalSearchResultToElasticEntity(0, result)
	if err != nil {
		return nil,
			fmt.Errorf(
				"GetById result unmarshall failed: id: %s error: %w",
				id.String(), err)
	}

	return data, e.validate.Struct(data)
}

// GetByHash implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) GetByHash(
	ctx context.Context,
	hash string,
) (*entity.ElasticImageMetaData, error) {
	slog.Info("GetByHash: call",
		"hash", hash)

	query := types.NewQuery()
	query.QueryString = types.NewQueryStringQuery()
	query.QueryString.Query = fmt.Sprintf("Hash: \"%s\"", hash)

	result, err := e.runSearchQuery(ctx, query, nil, nil)

	if err != nil {
		return nil, err
	}

	data, err := unmarhalSearchResultToElasticEntity(0, result)
	if err != nil {
		return nil,
			fmt.Errorf(
				"GetByHash result unmarshall failed: id: %s error: %w",
				hash, err)
	}

	if data == nil {
		return nil, nil
	}

	return data, e.validate.Struct(data)
}

// GetByPixels implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) GetByEmbeddingV1(
	ctx context.Context,
	img *entity.ElasticEmbeddingV1,
	count int,
) ([]*entity.ElasticImageMetaData, error) {
	slog.Info("GetByEmbeddingV1: call")

	query := e.embeddingV1KnnQuery(img, count)
	result, err := e.processKnn(ctx, *query)
	if err != nil {
		return nil, err
	}

	resultsSize := len(result.Hits.Hits)
	resultsEntity := make([]*entity.ElasticImageMetaData, 0)

	for index := range resultsSize {
		if float64(*(result.Hits.Hits[index].Score_)) < e.embeddingMatchTreshold {
			continue
		}

		item, err := unmarhalSearchResultToElasticEntity(index, result)
		if err != nil {
			return nil, fmt.Errorf("GetByEmbeddingV1 result unmarshall failed: error: %w", err)
		}

		err = e.validate.Struct(item)
		if err != nil {
			return nil, fmt.Errorf("GetByEmbeddingV1 result vaildation failed: error: %w", err)
		}

		resultsEntity = append(resultsEntity, item)
	}
	return resultsEntity, nil
}

// GetByHashAndAccountId implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) GetByHashAndAccountId(
	ctx context.Context,
	accountId uuid.UUID,
	hash string) (*entity.ElasticImageMetaData, error) {

	query := types.NewQuery()
	query.QueryString = types.NewQueryStringQuery()
	query.QueryString.Query = fmt.Sprintf(
		"Hash: \"%s\" AND AccountId: \"%s\"",
		hash, accountId.String())

	result, err := e.client.Search().
		Index(INDEX_NAME).
		Query(query).
		Do(ctx)

	if err != nil {
		return nil, err
	}

	data, err := unmarhalSearchResultToElasticEntity(0, result)
	if err != nil {
		return nil, err
	}

	return data, e.validate.Struct(data)
}

// Save implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) Save(ctx context.Context, file *entity.ElasticImageMetaData) error {

	file.Updated = time.Now().UnixMicro()

	buff := bytes.NewBuffer(nil)
	jsonEncoder := json.NewEncoder(buff)
	jsonEncoder.Encode(file)
	response, err := e.client.
		Index(INDEX_NAME).
		Document(file).
		Id(file.ImageId.String()).
		Do(ctx)

	if err != nil {
		return fmt.Errorf("Save metadata document error: id=%s : %w", file.ImageId, err)
	}

	slog.Info("Save metadata document",
		"id", file.ImageId)

	slog.Debug("Save metadata document details",
		"id", file.ImageId,
		"response", render.Render(response))

	return err
}

func unmarhalSearchResultToMatchedContent(i int, result *search.Response) (*entity.ElasticMatchedContent, error) {
	hits := result.Hits.Hits
	if len(hits) == 0 {
		return nil, nil
	}

	hit := hits[i]

	document, err := unmarhalSourceDocument(hit.Source_)
	if err != nil {
		return nil, err
	}
	highlights := hit.Highlight["Result"]

	matchedContent := new(entity.ElasticMatchedContent)
	matchedContent.Metadata = document
	matchedContent.ResultMatched = &highlights
	return matchedContent, nil
}

func unmarhalSearchResultToElasticEntity(i int, result *search.Response) (*entity.ElasticImageMetaData, error) {
	hits := result.Hits.Hits
	if len(hits) == 0 {
		return nil, nil
	}

	hit := hits[i]

	return unmarhalSourceDocument(hit.Source_)
}

func unmarhalSourceDocument(result json.RawMessage) (*entity.ElasticImageMetaData, error) {

	var document entity.ElasticImageMetaData
	err := json.Unmarshal(result, &document)
	return &document, err
}

func addr[T any](v T) *T { return &v }

func NewElasticMetadataStorage(
	config *conf.MetadataStorageConfig,
	validate *validator.Validate,
) MetadataStorageService {
	es8, _ := elasticsearch8.NewTypedClient(*config.Elastic)

	responseCreate, err := es8.Indices.
		Create(config.Index).
		Do(context.Background())

	slog.Info("Elastic create index",
		"index", config.Index,
		"response", render.Render(responseCreate),
		commonconst.ERR_LOG, err)

	indexTypeMapping := types.NewTypeMapping()
	indexTypeMapping.Properties["Created"] = types.NewLongNumberProperty()
	indexTypeMapping.Properties["Updated"] = types.NewLongNumberProperty()
	indexTypeMapping.Properties["AccountId"] = types.NewKeywordProperty()
	indexTypeMapping.Properties["Hash"] = types.NewKeywordProperty()
	indexTypeMapping.Properties["ImageId"] = types.NewKeywordProperty()

	denseProp := types.NewDenseVectorProperty()
	denseProp.Index = addr(true)
	denseProp.Dims = addr(config.EmbeddingV1Dimensions)
	indexTypeMapping.Properties["EmbeddingV1.Data"] = denseProp

	responseMapping, err := es8.Indices.PutMapping(config.Index).
		Properties(indexTypeMapping.Properties).
		Do(context.Background())

	slog.Info("Elastic create mapping index",
		"response", render.Render(responseMapping),
		commonconst.ERR_LOG, err)

	return &ElasticMetadataStorageServiceImpl{
		client:                 es8,
		embeddingMatchTreshold: config.EmbeddingMatchTreshold,
		validate:               validate,
	}
}
