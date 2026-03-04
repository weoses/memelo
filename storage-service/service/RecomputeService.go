package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
)

type ProgressDataRecompute struct {
	processed int
}

type RecomputeService interface {
	RecomputeOcrData(
		ctx context.Context,
		steps *StepsToDo,
		accountId *uuid.UUID,
		id *uuid.UUID,
		callback func(ctx context.Context, recompute ProgressDataRecompute) error) error
}

type RecomputeServiceImpl struct {
	slogger         *slog.Logger
	extractService  ImageMetadataExtractService
	metadataService MetadataStorageService
	imageService    ImageStorageService
}

func (r *RecomputeServiceImpl) RecomputeOcrData(
	ctx context.Context,
	steps *StepsToDo,
	accountId *uuid.UUID,
	id *uuid.UUID,
	callback func(ctx context.Context, recompute ProgressDataRecompute) error) error {
	pageSize := exportPageSize
	var afterId *uuid.UUID
	processed := 0

	for {
		page, err := r.metadataService.List(ctx, accountId, id, afterId, &pageSize)
		if err != nil {
			return fmt.Errorf("export: query metadata page failed: %w", err)
		}
		if len(page) == 0 {
			break
		}

		for _, meta := range page {
			if err := r.recomputeOne(ctx, steps, meta); err != nil {
				return fmt.Errorf("recompute: item %s failed: %w", meta.ImageId, err)
			}
			processed++
		}

		last := page[len(page)-1]
		afterId = &last.ImageId

		err = callback(ctx, ProgressDataRecompute{processed: processed})
		if err != nil {
			return fmt.Errorf("recompute: callback failed: %w", err)
		}

		if len(page) < pageSize {
			break
		}
	}

	return nil
}

func (r *RecomputeServiceImpl) recomputeOne(ctx context.Context, steps *StepsToDo, data *entity.ElasticImageMetaData) error {
	rawImg, err := r.imageService.GetImageBytes(ctx, data.S3Id)
	if err != nil {
		return fmt.Errorf("recompute: get image bytes failed: %w", err)
	}

	resultCtx, err := r.extractService.ProcessCreate(ctx,
		data.AccountId,
		steps,
		rawImg,
	)

	if err != nil {
		return fmt.Errorf("recompute: process create failed: %w", err)
	}

	if resultCtx.ImageHash != nil {
		data.Hash = *resultCtx.ImageHash
	}

	if resultCtx.ImageOcrResult != nil {
		data.Result = *resultCtx.ImageOcrResult
	}

	if resultCtx.ImageThumbnail != nil {
		err = r.imageService.Save(ctx, data.S3Id, rawImg, *resultCtx.ImageThumbnail)
		if err != nil {
			return fmt.Errorf("recompute: save image failed: %w", err)
		}
	}

	if resultCtx.ImageRawSize != nil {

	}
}

func NewRecomputeService(extractService ImageMetadataExtractService) RecomputeService {
	return &RecomputeServiceImpl{
		slogger:        slog.With("service", "RecomputeService"),
		extractService: extractService,
	}
}
