package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
)

type ExtractPipelineStep interface {
	GetPos() int
	Do(ctx context.Context, pipelineContext *ImageMetadataPipelineContext) error
}
type ImageMetadataPipelineContext struct {
	AccountId          uuid.UUID
	ImageHash          string
	ImageEmbedding     entity.ElasticEmbeddingV1
	ImageOcrResult     string
	ImageThumbnail     []byte
	ImageThumbnailSize entity.ElasticSizes
	ImageRaw           []byte
	ImageRawSize       entity.ElasticSizes
	Duplicate          *entity.ElasticImageMetaData
	Tags               []entity.ElasticTag
}
