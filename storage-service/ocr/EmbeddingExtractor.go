package ocr

import (
	"context"

	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/entity"
)

type EmbeddingExtractor interface {
	GetImageEmbedding(ctx context.Context, image temp.Data) (*entity.EmbeddingItem, error)
	GetTextEmbedding(ctx context.Context, text string) (*entity.EmbeddingItem, error)
	GetVideoEmbedding(ctx context.Context, video temp.Data) ([]*entity.EmbeddingItem, error)
}
