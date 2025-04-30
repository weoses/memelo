package service

import (
	"context"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"mine.local/ocr-gallery/storage-service/conf"
	"mine.local/ocr-gallery/storage-service/entity"
)

type MetadataStorageService interface {
	GetByHashAll(ctx context.Context, hash string) (*entity.ElasticImageMetaData, error)
	GetByEmbeddingV1All(ctx context.Context, img *entity.ElasticEmbeddingV1, count int) ([]*entity.ElasticImageMetaData, error)

	Save(ctx context.Context, file *entity.ElasticImageMetaData) error

	Search(ctx context.Context,
		accountId uuid.UUID,
		query string,
		sortIdAfter *int64,
		pageSize *int,
	) ([]*entity.ElasticMatchedContent, error)

	GetByHash(ctx context.Context, accountId uuid.UUID, hash string) (*entity.ElasticImageMetaData, error)
	GetById(ctx context.Context, accountId uuid.UUID, id uuid.UUID) (*entity.ElasticImageMetaData, error)

	DeleteById(ctx context.Context, accountId uuid.UUID, id uuid.UUID) error
}

func NewMetadataStorageService(config *conf.MetadataStorageConfig, validate *validator.Validate) MetadataStorageService {
	return NewElasticMetadataStorage(config, validate)
}
