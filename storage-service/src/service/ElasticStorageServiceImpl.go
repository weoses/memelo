package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"

	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/gdexlab/go-render/render"
	"mine.local/ocr-gallery/image-collector/conf"
	"mine.local/ocr-gallery/image-collector/entity"
)

const INDEX_NAME = "image-metadata"

type ElasticMetadataStorageServiceImpl struct {
	client *elasticsearch8.TypedClient
}

// Search implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) Search(ctx context.Context, queryString string) ([]*entity.ElasticImageMetaData, error) {
	query := types.NewQuery()
	query.Match = map[string]types.MatchQuery{
		"Result.Texts.Text": {
			Query:     queryString,
			Fuzziness: "AUTO",
		},
	}
	result, err := e.client.Search().
		Index(INDEX_NAME).
		Query(query).
		Do(ctx)

	if err != nil {
		return nil, err
	}

	results := make([]*entity.ElasticImageMetaData, len(result.Hits.Hits))

	for index, hit := range result.Hits.Hits {
		item, err := unmarhalSearchResultDocument(hit.Source_)
		if err != nil {
			return nil, err
		}
		results[index] = item
	}

	return results, nil
}

// GetById implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) GetById(ctx context.Context, id string) (*entity.ElasticImageMetaData, error) {
	query := types.NewQuery()
	query.Ids = types.NewIdsQuery()
	query.Ids.Values = []string{id}

	result, err := e.client.Search().
		Index(INDEX_NAME).
		Query(query).
		Do(ctx)

	if err != nil {
		return nil, err
	}

	return unmarhalSearchResult(result)
}

// GetByHash implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) GetByHash(
	ctx context.Context,
	hash string,
) (*entity.ElasticImageMetaData, error) {
	query := types.NewQuery()
	query.QueryString = types.NewQueryStringQuery()
	query.QueryString.Query = fmt.Sprintf("Storage.Hash: \"%s\"", hash)

	result, err := e.client.Search().
		Index(INDEX_NAME).
		Query(query).
		Do(ctx)

	if err != nil {
		return nil, err
	}

	return unmarhalSearchResult(result)
}

// Save implements MetadataStorageService.
func (e *ElasticMetadataStorageServiceImpl) Save(ctx context.Context, file *entity.ElasticImageMetaData) error {
	buff := bytes.NewBuffer(nil)
	jsonEncoder := json.NewEncoder(buff)
	jsonEncoder.Encode(file)
	response, err := e.client.
		Index(INDEX_NAME).
		Document(file).
		Id(file.Id).
		Do(ctx)

	if err != nil {
		log.Printf("Save metadata document error: id=%s error=%s", file.Id, err.Error())
		return err
	}

	log.Printf("Save metadata document: elastic, id=%s response=%s",
		file.Id, render.Render(response))

	return err
}

func unmarhalSearchResult(result *search.Response) (*entity.ElasticImageMetaData, error) {
	hits := result.Hits.Hits
	if len(hits) == 0 {
		return nil, nil
	}

	hit := hits[0]
	item := hit.Source_

	return unmarhalSearchResultDocument(item)
}

func unmarhalSearchResultDocument(result json.RawMessage) (*entity.ElasticImageMetaData, error) {

	var document entity.ElasticImageMetaData
	err := json.Unmarshal(result, &document)
	return &document, err
}

func NewElasticMetadataStorage(config *conf.MetadataStorageConfig) MetadataStorageService {
	es8, _ := elasticsearch8.NewTypedClient(*config.Elastic)
	response, err := es8.Indices.
		Create(config.Index).
		Do(context.TODO())

	log.Printf("Elastic create index response: %s error: %v", render.Render(response), err)

	return &ElasticMetadataStorageServiceImpl{
		client: es8,
	}
}
