package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/entity"
)

type ExtractPipelineStep interface {
	GetPos() int
	GetAllowedPipelineTypes() []entity.MetadataType
	Do(ctx context.Context, inputContext MetadataInputContext, pipelineContext *MetadataPipelineContext) error
}
type BasePipelineStep struct {
	pos int
	typ []entity.MetadataType
}

func (bp *BasePipelineStep) GetPos() int {
	return bp.pos
}

func (bp *BasePipelineStep) GetAllowedPipelineTypes() []entity.MetadataType {
	return bp.typ
}

type MetadataInputContext struct {
	RawInput  temp.S3BackedData
	AccountId uuid.UUID
	Type      entity.MetadataType
}

type MetadataStorageArtifact struct {
	Type SavedArtifactType
	Data temp.S3BackedData
}

func (m *MetadataStorageArtifact) Close() error {
	if m.Data != nil {
		tmpLogger := slog.With("service", "MetadataStorageArtifact")
		helper.QuietClose(m.Data, tmpLogger)
	}
	return nil
}

type MetadataPipelineContext struct {
	Hash             string
	Embedding        []entity.EmbeddingItem
	Result           *entity.Result
	StorageArtifacts []MetadataStorageArtifact
	Duplicate        *entity.ElasticImageMetaData
	Tags             []entity.ElasticTag

	ImageOriginalJpeg temp.S3BackedData
	VideoMp4          temp.S3BackedData
	ImageThumbnail    temp.S3BackedData

	OriginalSize  entity.Sizes
	ThumbnailSize entity.Sizes
}

func (m *MetadataPipelineContext) Close() error {
	tmpLogger := slog.With("service", "MetadataPipelineContext")
	if m.StorageArtifacts != nil {
		for i := range m.StorageArtifacts {
			helper.QuietClose(&m.StorageArtifacts[i], tmpLogger)
		}
	}

	helper.QuietClose(m.ImageOriginalJpeg, tmpLogger)
	helper.QuietClose(m.ImageThumbnail, tmpLogger)
	helper.QuietClose(m.VideoMp4, tmpLogger)

	return nil
}
