package service

import (
	"context"

	"mine.local/ocr-gallery/image-collector/conf"
	"mine.local/ocr-gallery/image-collector/entity"
)

type MetadataStorageService interface {
	Save(ctx context.Context, file *entity.ElasticImageMetaData) error
	GetByHash(ctx context.Context, hash string) (*entity.ElasticImageMetaData, error)
	GetById(ctx context.Context, id string) (*entity.ElasticImageMetaData, error)
	Search(ctx context.Context, query string) ([]*entity.ElasticImageMetaData, error)

	//Delete(ctx context.Context, id string) error
	//Exists(ctx context.Context, id string) (bool, error)
}

func NewMetadataStorageService(config *conf.MetadataStorageConfig) MetadataStorageService {
	return NewElasticMetadataStorage(config)
}
