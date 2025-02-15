package service

import (
	"context"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"mine.local/ocr-gallery/storage-service/conf"
	"mine.local/ocr-gallery/storage-service/entity"
)

type MetadataStorageService interface {
	Save(ctx context.Context, file *entity.ElasticImageMetaData) error
	GetByHash(ctx context.Context, hash string) (*entity.ElasticImageMetaData, error)
	GetByEmbeddingV1(ctx context.Context, img entity.ElasticEmbeddingV1) (*entity.ElasticImageMetaData, error)
	GetByHashAndAccountId(ctx context.Context, accountId uuid.UUID, hash string) (*entity.ElasticImageMetaData, error)
	GetById(ctx context.Context, id uuid.UUID) (*entity.ElasticImageMetaData, error)
	Search(ctx context.Context,
		accountId uuid.UUID,
		query string,
		sortIdAfter *int64,
		pageSize *int,
	) ([]*entity.ElasticMatchedContent, error)

	//Delete(ctx context.Context, id string) error
	//Exists(ctx context.Context, id string) (bool, error)
}

func NewMetadataStorageService(config *conf.MetadataStorageConfig, validate *validator.Validate) MetadataStorageService {
	return NewElasticMetadataStorage(config, validate)
}
