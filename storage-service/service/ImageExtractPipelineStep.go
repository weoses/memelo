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
	GetKey() string
	GetAllowedPipelineTypes() []entity.MetadataType
	Do(ctx context.Context, inputContext MetadataInputContext, pipelineContext *MetadataPipelineContext) error
}
type BasePipelineStep struct {
	pos int
	typ []entity.MetadataType
	key string
}

func (bp *BasePipelineStep) GetKey() string {
	return bp.key
}

func (bp *BasePipelineStep) GetPos() int {
	return bp.pos
}

func (bp *BasePipelineStep) GetAllowedPipelineTypes() []entity.MetadataType {
	return bp.typ
}

type StepCallback func(ctx context.Context, stepName string, pipelineContext *MetadataPipelineContext) bool
type MetadataInputContext struct {
	RawInput       temp.S3BackedData
	AccountId      uuid.UUID
	Type           entity.MetadataType
	StepCallback   StepCallback
	SeedImageId    *uuid.UUID
	SeedEmbeddings []entity.EmbeddingItem
}

type MetadataStorageArtifact struct {
	Type SavedArtifactType
	Data temp.S3BackedData
}

func (m MetadataStorageArtifact) Close(ctx context.Context) error {
	if m.Data != nil {
		tmpLogger := slog.With("service", "MetadataStorageArtifact")
		helper.QuietCloseCtx(ctx, m.Data, tmpLogger)
	}
	return nil
}

type VideoSlice struct {
	SliceNumber    int
	SliceStartTime int
	SliceEndTime   int
	Slice          temp.S3BackedData
}

func (vs VideoSlice) Close(ctx context.Context) error {
	return vs.Slice.Close(ctx)
}

type MetadataPipelineContext struct {
	Hash                 string
	Embedding            []entity.EmbeddingItem
	Result               *entity.Result
	ResultPerVideoSlices []entity.ResultPerVideoSlice
	StorageArtifacts     []MetadataStorageArtifact
	Duplicate            *entity.ElasticImageMetaData
	Tags                 []entity.ElasticTag

	ImageOriginalJpeg temp.S3BackedData
	VideoMp4          temp.S3BackedData
	VideoSlices       []VideoSlice
	ImageThumbnail    temp.S3BackedData

	OriginalSize  entity.Sizes
	ThumbnailSize entity.Sizes
}

func (m *MetadataPipelineContext) Close(ctx context.Context) error {
	tmpLogger := slog.With("service", "MetadataPipelineContext")

	helper.QuietCloseCtx(ctx, m.ImageOriginalJpeg, tmpLogger)
	helper.QuietCloseCtx(ctx, m.ImageThumbnail, tmpLogger)
	helper.QuietCloseCtx(ctx, m.VideoMp4, tmpLogger)
	helper.QuietCloseAllCtx(ctx, m.VideoSlices, tmpLogger)
	helper.QuietCloseAllCtx(ctx, m.StorageArtifacts, tmpLogger)

	return nil
}
