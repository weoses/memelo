package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
)

type ExtractPipelineStep interface {
	GetPos() int
	Check(ctx context.Context, steps *StepsToDo) (bool, error)
	Do(ctx context.Context, pipelineContext *ImageMetadataPipelineContext) error
}
type ImageMetadataPipelineContext struct {
	AccountId          uuid.UUID
	ImageHash          *string
	ImageEmbedding     *entity.ElasticEmbeddingV1
	ImageOcrResult     *string
	ImageThumbnail     *[]byte
	ImageThumbnailSize *entity.ElasticSizes
	ImageRaw           []byte
	ImageRawSize       *entity.ElasticSizes
	Duplicate          *entity.ElasticImageMetaData
}

type StepsToDo struct {
	DuplicateSearch bool
	Ocr             bool
	CreateThumbnail bool
	CalcSize        bool
	CreateEmbedding bool
	CalcHash        bool
}
