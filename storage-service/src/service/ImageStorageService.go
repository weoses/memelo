package service

import (
	"context"

	"github.com/google/uuid"
	"mine.local/ocr-gallery/storage-service/conf"
	"mine.local/ocr-gallery/storage-service/entity"
)

type ImageStorageService interface {
	Save(ctx context.Context, id uuid.UUID, image *entity.Image, thumb *entity.Image) error
	GetImage(ctx context.Context, id uuid.UUID) (*entity.Image, error)

	GetUrl(ctx context.Context, id uuid.UUID) (string, error)
	GetUrlThumb(ctx context.Context, id uuid.UUID) (string, error)
}

func NewImageStorageService(config *conf.ImageStorageConfig) (ImageStorageService, error) {
	return NewMinioFileStorageServiceImpl(config)
}
