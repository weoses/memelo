package ocr

import (
	"context"
	"log/slog"

	"github.com/bluele/gcache"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/entity"
)

const defaultTextEmbeddingCacheSize = 512

type CachedEmbeddingExtractor struct {
	inner   LlmEmbeddingExtractor
	cache   gcache.Cache
	sLogger *slog.Logger
}

func NewCachedEmbeddingExtractor(inner LlmEmbeddingExtractor) LlmEmbeddingExtractor {
	c := gcache.New(defaultTextEmbeddingCacheSize).LFU().Build()
	return &CachedEmbeddingExtractor{inner: inner, cache: c, sLogger: slog.With("service", "CachedEmbeddingExtractor")}
}

func (c *CachedEmbeddingExtractor) GetTextEmbedding(ctx context.Context, text string) (*entity.EmbeddingItem, error) {
	if v, err := c.cache.Get(text); err == nil {
		c.sLogger.InfoContext(ctx, "GetTextEmbedding: cache hit", "text", text)
		return v.(*entity.EmbeddingItem), nil
	}
	result, err := c.inner.GetTextEmbedding(ctx, text)
	if err != nil {
		return nil, err
	}
	_ = c.cache.Set(text, result)
	return result, nil
}

func (c *CachedEmbeddingExtractor) GetImageEmbedding(ctx context.Context, image temp.Data) (*entity.EmbeddingItem, error) {
	return c.inner.GetImageEmbedding(ctx, image)
}

func (c *CachedEmbeddingExtractor) GetVideoEmbedding(ctx context.Context, video temp.Data) ([]entity.EmbeddingItem, error) {
	return c.inner.GetVideoEmbedding(ctx, video)
}
