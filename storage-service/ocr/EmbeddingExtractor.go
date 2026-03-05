package ocr

import (
	"context"

	"github.com/weoses/memelo/storage-service/entity"
)

type EmbeddingExtractor interface {
	GetImageEmbeddingV1(ctx context.Context, image []byte) (*entity.ElasticEmbeddingV1, error)
	GetTextEmbeddingV1(ctx context.Context, text string) (*entity.ElasticEmbeddingV1, error)
}
