package service

import (
	"context"

	"mine.local/ocr-gallery/image-collector/conf"
)

type ImageStorageService interface {
	Save(ctx context.Context, base64Image *string) (string, error)
	//Delete(ctx context.Context, id string)
	Get(ctx context.Context, id string) (*string, error)
}

func NewImageStorageService(config *conf.ImageStorageConfig) ImageStorageService {
	return NewFilesystemImageStorageService(config)
}
