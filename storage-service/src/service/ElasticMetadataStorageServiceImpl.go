package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/operator"
	"github.com/gdexlab/go-render/render"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"mine.local/ocr-gallery/storage-service/conf"
	"mine.local/ocr-gallery/storage-service/entity"
)

const INDEX_NAME = "image-metadata"

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
		return err
	}

	log.Printf("Delete metadata document: elastic, id=%s response=%s", id, render.Render(response))
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

	return search.Do(ctx)
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

	return search.Do(ctx)
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
		log.Printf("Search using ALL query: query: '%s' account: '%s' after: %v",
			queryString, accountId, sortIdAfter)
		q := e.allQuery(accountId)
		return e.runSearchQuery(ctx, q, sortIdAfter, pageSize)
	}

	log.Printf("Search using simple query: query: '%s' account: '%s' after: %v",
		queryString, accountId, sortIdAfter)
	q := e.simpleQuery(accountId, queryString)
	res, err := e.runSearchQuery(ctx, q, sortIdAfter, pageSize)
	if err != nil {
		return nil, err
	}

	resultsSize := len(res.Hits.Hits)
	if resultsSize > 0 || sortIdAfter != nil {
		return res, nil
	}

	log.Printf("Search using fuzzy query: query: '%s' account: '%s'", queryString, accountId)
	q = e.fuzzySearchQuery(accountId, queryString)
	return e.runSearchQuery(ctx, q, sortIdAfter, pageSize)

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
		return nil, err
	}

	resultsSize := len(result.Hits.Hits)
	results := make([]*entity.ElasticMatchedContent, resultsSize)

	for index := range resultsSize {
		item, err := unmarhalSearchResultToMatchedContent(index, result)
		if err != nil {
			return nil, err
		}
		err = e.validate.Struct(item)
		if err != nil {
			return nil, err
		}
		results[index] = item
	}
	return results, nil
}

// GetById implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) GetById(ctx context.Context, id uuid.UUID) (*entity.ElasticImageMetaData, error) {
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
		return nil, err
	}

	return data, e.validate.Struct(data)
}

// GetByHash implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) GetByHash(
	ctx context.Context,
	hash string,
) (*entity.ElasticImageMetaData, error) {
	query := types.NewQuery()
	query.QueryString = types.NewQueryStringQuery()
	query.QueryString.Query = fmt.Sprintf("Hash: \"%s\"", hash)

	result, err := e.runSearchQuery(ctx, query, nil, nil)

	if err != nil {
		return nil, err
	}

	data, err := unmarhalSearchResultToElasticEntity(0, result)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		err = e.validate.Struct(item)
		if err != nil {
			return nil, err
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
		log.Printf("Save metadata document error: id=%s error=%s", file.ImageId, err.Error())
		return err
	}

	log.Printf("Save metadata document: elastic, id=%s response=%s",
		file.ImageId, render.Render(response))

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
	log.Printf("Elastic create index response: %s error: %v", render.Render(responseCreate), err)

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

	log.Printf("Elastic mapping index response: %s error: %v", render.Render(responseMapping), err)

	return &ElasticMetadataStorageServiceImpl{
		client:                 es8,
		embeddingMatchTreshold: config.EmbeddingMatchTreshold,
		validate:               validate,
	}
}
