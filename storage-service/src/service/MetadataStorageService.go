package service

import (
	"context"

	"github.com/google/uuid"
	"mine.local/ocr-gallery/storage-service/conf"
	"mine.local/ocr-gallery/storage-service/entity"
)

type MetadataStorageService interface {
	Save(ctx context.Context, file *entity.ElasticImageMetaData) error
	GetByHash(ctx context.Context, hash string) (*entity.ElasticImageMetaData, error)
	GetByHashAndAccountId(ctx context.Context, accountId uuid.UUID, hash string) (*entity.ElasticImageMetaData, error)
	GetById(ctx context.Context, id uuid.UUID) (*entity.ElasticImageMetaData, error)
	Search(ctx context.Context, accountId uuid.UUID, query string) ([]*entity.ElasticMatchedContent, error)

	//Delete(ctx context.Context, id string) error
	//Exists(ctx context.Context, id string) (bool, error)
}

func NewMetadataStorageService(config *conf.MetadataStorageConfig) MetadataStorageService {
	return NewElasticMetadataStorage(config)
}
