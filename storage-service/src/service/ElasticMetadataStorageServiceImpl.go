package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/gdexlab/go-render/render"
	"github.com/google/uuid"
	"mine.local/ocr-gallery/storage-service/conf"
	"mine.local/ocr-gallery/storage-service/entity"
)

const INDEX_NAME = "image-metadata"

type ElasticMetadataStorageServiceImpl struct {
	client *elasticsearch8.TypedClient
}

// Search implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) Search(
	ctx context.Context,
	accountId uuid.UUID,
	queryString string,
	searchAfter *uuid.UUID,
	pageSize *int,
) ([]*entity.ElasticMatchedContent, error) {
	q1 := types.NewQuery()
	q1.Match = map[string]types.MatchQuery{
		"Result": {
			Query:     queryString,
			Fuzziness: "AUTO",
		},
	}

	q2 := types.NewQuery()
	q2.Match = map[string]types.MatchQuery{
		"AccountId": {
			Query:     accountId.String(),
			Fuzziness: 0,
		},
	}

	query := types.NewQuery()
	query.Bool = types.NewBoolQuery()
	query.Bool.Must = []types.Query{
		*q1, *q2,
	}

	highlight := types.NewHighlight()
	highlight.PreTags = []string{"<MATCH>"}
	highlight.PostTags = []string{"</MATCH>"}
	highlight.Fields = map[string]types.HighlightField{
		"Result": *types.NewHighlightField(),
	}

	resultField := types.NewFieldAndFormat()
	resultField.Field = "Result"

	sortCreated := types.NewSortOptions()
	sortCreated.SortOptions["Created"] = *types.NewFieldSort()

	sortId := types.NewSortOptions()
	sortId.SortOptions["Id"] = *types.NewFieldSort()

	search := e.client.Search().
		Index(INDEX_NAME).
		Query(query).
		Fields(*resultField).
		Highlight(highlight).
		Sort(sortCreated, sortId)

	if searchAfter != nil {
		search = search.SearchAfter(searchAfter.String())
	}

	if pageSize != nil {
		search = search.Size(*pageSize)
	}

	result, err := search.Do(ctx)
	if err != nil {
		return nil, err
	}

	resultsSize := len(result.Hits.Hits)

	results := make([]*entity.ElasticMatchedContent, resultsSize)

	for index := range resultsSize {
		item, err := unmarhalSearchResult(index, result)
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

	return unmarhalSearchResultToElasticEntity(0, result)
}

// GetByHash implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) GetByHash(
	ctx context.Context,
	hash string,
) (*entity.ElasticImageMetaData, error) {
	query := types.NewQuery()
	query.QueryString = types.NewQueryStringQuery()
	query.QueryString.Query = fmt.Sprintf("Hash: \"%s\"", hash)

	result, err := e.client.Search().
		Index(INDEX_NAME).
		Query(query).
		Do(ctx)

	if err != nil {
		return nil, err
	}

	return unmarhalSearchResultToElasticEntity(0, result)
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

	return unmarhalSearchResultToElasticEntity(0, result)
}

// Save implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) Save(ctx context.Context, file *entity.ElasticImageMetaData) error {

	file.Created = time.Now().UnixMicro()

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

func unmarhalSearchResult(i int, result *search.Response) (*entity.ElasticMatchedContent, error) {
	hits := result.Hits.Hits
	if len(hits) == 0 {
		return nil, nil
	}

	hit := hits[i]

	document, err := unmarhalSearchResultDocument(hit.Source_)
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

	return unmarhalSearchResultDocument(hit.Source_)
}

func unmarhalSearchResultDocument(result json.RawMessage) (*entity.ElasticImageMetaData, error) {

	var document entity.ElasticImageMetaData
	err := json.Unmarshal(result, &document)
	return &document, err
}

func NewElasticMetadataStorage(config *conf.MetadataStorageConfig) MetadataStorageService {
	es8, _ := elasticsearch8.NewTypedClient(*config.Elastic)

	indexTypeMapping := types.NewTypeMapping()

	indexTypeMapping.Properties["Created"] = types.NewLongNumberProperty()
	indexTypeMapping.Properties["AccountId"] = types.NewKeywordProperty()
	indexTypeMapping.Properties["Hash"] = types.NewKeywordProperty()
	indexTypeMapping.Properties["Id"] = types.NewKeywordProperty()

	response, err := es8.Indices.
		Create(config.Index).
		Mappings(indexTypeMapping).
		Do(context.Background())

	log.Printf("Elastic create index response: %s error: %v", render.Render(response), err)

	return &ElasticMetadataStorageServiceImpl{
		client: es8,
	}
}
