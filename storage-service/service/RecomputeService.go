package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/storage-service/entity"
	storage2 "github.com/weoses/memelo/storage-service/storage"
)

type ProgressDataRecompute struct {
	Processed int
}

type RecomputeService interface {
	Recompute(
		ctx context.Context,
		accountId *uuid.UUID,
		id *uuid.UUID,
		callback func(ctx context.Context, recompute ProgressDataRecompute) error) error
}

type RecomputeServiceImpl struct {
	slogger                *slog.Logger
	extractService         ImageMetadataExtractService
	metadataStorageService storage2.MetadataStorageService
	imageStorageService    storage2.ImageStorageService
}

func (r *RecomputeServiceImpl) Recompute(
	ctx context.Context,
	accountId *uuid.UUID,
	id *uuid.UUID,
	callback func(ctx context.Context, recompute ProgressDataRecompute) error) error {
	pageSize := exportPageSize
	var afterId *uuid.UUID
	processed := 0

	for {
		page, err := r.metadataStorageService.List(ctx, accountId, id, afterId, &pageSize)
		if err != nil {
			return fmt.Errorf("export: query metadata page failed: %w", err)
		}
		if len(page) == 0 {
			break
		}

		for _, meta := range page {
			if err = r.recomputeOne(ctx, meta); err != nil {
				r.slogger.Error("recompute: item failed:",
					"try", meta.ImageId,
					"imageId", meta.ImageId,
					"error", err)
				continue
			}
			processed++
		}

		last := page[len(page)-1]
		afterId = &last.ImageId

		err = callback(ctx, ProgressDataRecompute{Processed: processed})
		if err != nil {
			return fmt.Errorf("recompute: callback failed: %w", err)
		}

		if len(page) < pageSize {
			break
		}
	}
	r.slogger.Info("recompute finished:", "processed", processed)
	return nil
}

func (r *RecomputeServiceImpl) recomputeOne(ctx context.Context, data *entity.ElasticImageMetaData) error {
	rawImg, err := r.imageStorageService.GetImageBytes(ctx, data.S3Id)
	if err != nil {
		return fmt.Errorf("recompute: get image bytes failed: %w", err)
	}

	resultCtx, err := r.extractService.ProcessCreate(ctx,
		data.AccountId,
		rawImg,
		false,
	)

	if err != nil {
		return fmt.Errorf("recompute: process create failed: %w", err)
	}
	data.Hash = resultCtx.ImageHash
	data.Result = resultCtx.ImageOcrResult
	if resultCtx.ImageRaw != nil && resultCtx.ImageThumbnail != nil {
		err = r.imageStorageService.Save(ctx, data.S3Id, resultCtx.ImageRaw, resultCtx.ImageThumbnail)
		if err != nil {
			return fmt.Errorf("recompute: save image failed: %w", err)
		}
	}

	data.ImageSize = &entity.ElasticSizes{
		Width:  resultCtx.ImageRawSize.Width,
		Height: resultCtx.ImageRawSize.Height,
	}

	data.ThumbSize = &entity.ElasticSizes{
		Width:  resultCtx.ImageThumbnailSize.Width,
		Height: resultCtx.ImageThumbnailSize.Height,
	}

	data.EmbeddingV1 = &resultCtx.ImageEmbedding
	data.Tags = helper.TransformSlice(
		resultCtx.Tags,
		make([]string, len(resultCtx.Tags)),
		func(tag entity.ElasticTag) string {
			return tag.Tag
		})

	err = r.metadataStorageService.Save(ctx, data)
	if err != nil {
		return fmt.Errorf("recompute: save metadata failed: %w", err)
	}
	return nil
}

func NewRecomputeService(
	extractService ImageMetadataExtractService,
	metadataStorageService storage2.MetadataStorageService,
	imageStorageService storage2.ImageStorageService,
) RecomputeService {
	return &RecomputeServiceImpl{
		slogger:                slog.With("service", "RecomputeService"),
		extractService:         extractService,
		metadataStorageService: metadataStorageService,
		imageStorageService:    imageStorageService,
	}
}
