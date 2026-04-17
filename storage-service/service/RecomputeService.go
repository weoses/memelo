package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/service"
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
	extractService         MetadataExtractService
	metadataStorageService storage2.MetadataStorageService
	imageStorageService    storage2.MediaStorageService
	tmpDataService         service.TmpDataService
}

func (r *RecomputeServiceImpl) Recompute(
	ctx context.Context,
	accountId *uuid.UUID,
	id *uuid.UUID,
	callback func(ctx context.Context, recompute ProgressDataRecompute) error) error {
	processed := 0

	if id != nil && accountId != nil {
		item, err := r.metadataStorageService.GetById(ctx, *accountId, *id)
		if err != nil {
			return fmt.Errorf("recompute: query metadataService item failed: %w", err)
		}
		if item != nil {
			if err = r.recomputeOne(ctx, item); err != nil {
				r.slogger.Error("recompute: item failed:",
					"imageId", item.ImageId,
					"error", err)
			} else {
				processed++
			}
		}
		return callback(ctx, ProgressDataRecompute{Processed: processed})
	}

	var sortKey entity.ElasticSortKey

	for {
		var page []*entity.ElasticImageMetaData
		var nextKey entity.ElasticSortKey
		var err error

		if accountId != nil {
			page, nextKey, err = r.metadataStorageService.GetByAccountIdOrderByCreated(ctx, *accountId, sortKey, exportPageSize)
		} else {
			page, nextKey, err = r.metadataStorageService.GetAll(ctx, sortKey, exportPageSize)
		}
		if err != nil {
			return fmt.Errorf("recompute: query metadataService page failed: %w", err)
		}
		if len(page) == 0 {
			break
		}

		for _, meta := range page {
			if err = r.recomputeOne(ctx, meta); err != nil {
				r.slogger.Error("recompute: item failed:",
					"imageId", meta.ImageId,
					"error", err)
				continue
			}
			processed++
		}

		if err = callback(ctx, ProgressDataRecompute{Processed: processed}); err != nil {
			return fmt.Errorf("recompute: callback failed: %w", err)
		}

		if len(page) < exportPageSize {
			break
		}
		sortKey = nextKey
	}

	r.slogger.Info("recompute finished:", "processed", processed)
	return nil
}

func (r *RecomputeServiceImpl) recomputeOne(ctx context.Context, data *entity.ElasticImageMetaData) error {
	rawImg, err := r.imageStorageService.Read(ctx, data.S3Id, storageMediaType(data.Type, SavedOriginal))
	defer helper.QuietClose(rawImg, r.slogger)
	if err != nil {
		return fmt.Errorf("recompute: get image bytes failed: %w", err)
	}

	mime := "image/jpeg"
	if data.Type == entity.VideoMetadataType {
		mime = "video/mp4"
	}

	rawImgS3Backed, err := r.tmpDataService.WrapData(ctx, mime, rawImg)
	defer helper.QuietClose(rawImgS3Backed, r.slogger)
	if err != nil {
		return fmt.Errorf("recompute: wrap data failed: %w", err)
	}

	pipelineResult, err := r.extractService.Extract(ctx,
		data.AccountId,
		data.Type,
		rawImgS3Backed,
		false,
	)
	if err != nil {
		return fmt.Errorf("recompute: process create failed: %w", err)
	}
	defer helper.QuietClose(pipelineResult, r.slogger)

	data.Hash = pipelineResult.Hash

	joinedResult := fmt.Sprintf("%s %s %s",
		pipelineResult.Result.OnScreenText,
		pipelineResult.Result.AudioTranscript,
		pipelineResult.Result.AudioTrack)
	data.Result = joinedResult
	data.ResultData = pipelineResult.Result
	data.ResultPerVideoSlices = pipelineResult.ResultPerVideoSlices

	for i := range pipelineResult.StorageArtifacts {
		artifact := pipelineResult.StorageArtifacts[i]
		err = r.imageStorageService.Save(ctx, data.S3Id, storageMediaType(data.Type, artifact.Type), artifact.Data)
		if err != nil {
			return fmt.Errorf("save artifact with type %s failed: %w", artifact.Type, err)
		}
	}

	data.ImageSize = &entity.Sizes{
		Width:  pipelineResult.OriginalSize.Width,
		Height: pipelineResult.OriginalSize.Height,
	}

	data.ThumbSize = &entity.Sizes{
		Width:  pipelineResult.ThumbnailSize.Width,
		Height: pipelineResult.ThumbnailSize.Height,
	}

	data.EmbeddingList = pipelineResult.Embedding
	data.Tags = helper.TransformSlice(
		pipelineResult.Tags,
		make([]string, len(pipelineResult.Tags)),
		func(tag entity.ElasticTag) string {
			return tag.Tag
		})

	err = r.metadataStorageService.Save(ctx, data)
	if err != nil {
		return fmt.Errorf("recompute: save metadataService failed: %w", err)
	}
	return nil
}

func NewRecomputeService(
	extractService MetadataExtractService,
	metadataStorageService storage2.MetadataStorageService,
	imageStorageService storage2.MediaStorageService,
	tmpDataService service.TmpDataService,
) RecomputeService {
	return &RecomputeServiceImpl{
		slogger:                slog.With("service", "RecomputeService"),
		extractService:         extractService,
		metadataStorageService: metadataStorageService,
		imageStorageService:    imageStorageService,
		tmpDataService:         tmpDataService,
	}
}
