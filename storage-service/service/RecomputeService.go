package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/storage-service/entity"
	storage2 "github.com/weoses/memelo/storage-service/storage"
)

const maxRecomputeWorkers = 4
const recomputePageSize = 50

type RecomputeParams struct {
	Query              map[string]interface{}
	ComputeExtractor   bool
	ComputeEmbedding   bool
	UpdateStorageItems bool
	CheckDuplicates    bool

	IncludeManualEdited bool
}

type RecomputeService interface {
	StartRecompute(ctx context.Context, params RecomputeParams) (string, error)
	GetJobStatus(ctx context.Context, jobId string) (*RecomputeJobState, error)
	RecomputeOneById(ctx context.Context, accountId string, id string, params RecomputeParams) error
}

type RecomputeServiceImpl struct {
	slogger                *slog.Logger
	extractService         MetadataExtractService
	metadataStorageService storage2.MetadataStorageService
	imageStorageService    storage2.MediaStorageService
	tmpDataService         service.TmpDataService
	jobStorage             RecomputeJobStorage
}

func (r *RecomputeServiceImpl) StartRecompute(ctx context.Context, params RecomputeParams) (string, error) {
	jobId := uuid.New().String()
	job := r.jobStorage.Create(jobId)

	go r.runJob(context.WithoutCancel(ctx), job, params)

	return jobId, nil
}

func (r *RecomputeServiceImpl) GetJobStatus(_ context.Context, jobId string) (*RecomputeJobState, error) {
	job, ok := r.jobStorage.Get(jobId)
	if !ok {
		return nil, fmt.Errorf("job not found: %s", jobId)
	}
	return job, nil
}

func (r *RecomputeServiceImpl) runJob(ctx context.Context, job *RecomputeJobState, params RecomputeParams) {
	job.Mu.Lock()
	job.State = RecomputeStateRunning
	job.Mu.Unlock()

	rawQuery := params.Query
	if rawQuery == nil {
		rawQuery = map[string]interface{}{"match_all": map[string]interface{}{}}
	}

	var sortKey entity.ElasticSortKey
	sem := make(chan struct{}, maxRecomputeWorkers)
	var wg sync.WaitGroup

	for {
		page, nextKey, err := r.metadataStorageService.QueryByRaw(ctx, rawQuery, sortKey, recomputePageSize)
		if err != nil {
			r.slogger.ErrorContext(ctx, "recompute: pagination failed", "jobId", job.JobId, "error", err)
			job.Mu.Lock()
			job.State = RecomputeStateFailed
			job.Errors = append(job.Errors, RecomputeJobError{ErrorText: err.Error()})
			job.Mu.Unlock()
			return
		}
		if len(page) == 0 {
			break
		}

		for _, meta := range page {
			m := meta
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()

				r.slogger.InfoContext(ctx, "recompute: processing media object",
					"jobId", job.JobId,
					"imageId", m.ImageId)

				if err := r.recomputeOne(ctx, m, params); err != nil {
					r.slogger.ErrorContext(ctx, "recompute: object failed",
						"jobId", job.JobId,
						"imageId", m.ImageId,
						"error", err)
					job.Mu.Lock()
					job.Errors = append(job.Errors, RecomputeJobError{
						ObjectId:  m.ImageId.String(),
						ErrorText: err.Error(),
					})
					job.Mu.Unlock()
					return
				}

				job.Mu.Lock()
				job.Processed++
				job.LastId = m.ImageId.String()
				job.Mu.Unlock()
			}()
		}

		if len(page) < recomputePageSize {
			break
		}
		sortKey = nextKey
	}

	wg.Wait()

	job.Mu.Lock()
	job.State = RecomputeStateDone
	job.Mu.Unlock()

	r.slogger.InfoContext(ctx, "recompute: job done", "jobId", job.JobId, "processed", job.Processed)
}

func (r *RecomputeServiceImpl) recomputeOne(ctx context.Context, data *entity.ElasticImageMetaData, params RecomputeParams) error {
	if (!params.IncludeManualEdited) && data.Edited {
		r.slogger.InfoContext(ctx, "recompute: skipping manually-updated entity", "imageId", data.ImageId)
		return nil
	}

	rawImg, err := r.imageStorageService.Read(ctx, data.S3Id, storageMediaType(data.Type, SavedOriginal))
	defer helper.QuietClose(rawImg, r.slogger)
	if err != nil {
		return fmt.Errorf("get image bytes failed: %w", err)
	}

	mime := "image/jpeg"
	if data.Type == entity.VideoMetadataType {
		mime = "video/mp4"
	}

	rawImgS3Backed, err := r.tmpDataService.WrapData(ctx, mime, rawImg)
	defer helper.QuietClose(rawImgS3Backed, r.slogger)
	if err != nil {
		return fmt.Errorf("wrap data failed: %w", err)
	}

	skipStepsMap := map[string]bool{
		VidSliceKey:                  params.ComputeEmbedding || params.ComputeExtractor,
		ImgCalcEmbeddingKey:          params.ComputeEmbedding,
		VidCalcEmbeddingsKey:         params.ComputeEmbedding,
		ImgLlmExtractKey:             params.ComputeExtractor,
		VidLlmExtractKey:             params.ComputeExtractor,
		CheckDuplicateByHashKey:      params.CheckDuplicates,
		CheckDuplicateByEmbeddingKey: params.CheckDuplicates,
	}

	pipelineResult, err := r.extractService.Extract(ctx, MetadataInputContext{
		AccountId: data.AccountId,
		Type:      data.Type,
		RawInput:  rawImgS3Backed,
		StepCallback: func(ctx context.Context, stepName string, pipelineCtx *MetadataPipelineContext) bool {
			i, ok := skipStepsMap[stepName]
			if !ok {
				return true
			}
			return i
		},
		SeedImageId:    &data.ImageId,
		SeedEmbeddings: data.EmbeddingList,
	})

	if err != nil {
		return fmt.Errorf("extract pipeline failed: %w", err)
	}

	defer helper.QuietClose(pipelineResult, r.slogger)

	if params.CheckDuplicates && pipelineResult.Duplicate != nil {
		r.slogger.InfoContext(ctx, "recompute: deleting duplicate entity",
			"duplicateImageId", pipelineResult.Duplicate.ImageId)

		errMetadataDelete := r.metadataStorageService.DeleteById(ctx, data.AccountId, pipelineResult.Duplicate.ImageId)
		errOrigDelete := r.imageStorageService.Delete(ctx, data.S3Id, storageMediaType(data.Type, SavedOriginal))
		errThumbDelete := r.imageStorageService.Delete(ctx, data.S3Id, storageMediaType(data.Type, SavedThumb))

		if errors.Join(errMetadataDelete, errOrigDelete, errThumbDelete) != nil {
			return fmt.Errorf("delete duplicate failed: %w", err)
		}
	}

	data.Hash = pipelineResult.Hash

	if params.ComputeExtractor && pipelineResult.Result != nil {
		joinedResult := fmt.Sprintf("%s %s %s",
			pipelineResult.Result.OnScreenText,
			pipelineResult.Result.AudioTranscript,
			pipelineResult.Result.AudioTrack)
		data.Result = joinedResult
		data.ResultData = pipelineResult.Result
		data.ResultPerVideoSlices = pipelineResult.ResultPerVideoSlices
	}

	if params.UpdateStorageItems {
		for i := range pipelineResult.StorageArtifacts {
			artifact := pipelineResult.StorageArtifacts[i]
			err = r.imageStorageService.Save(ctx, data.S3Id, storageMediaType(data.Type, artifact.Type), artifact.Data)
			if err != nil {
				return fmt.Errorf("save artifact with type %s failed: %w", artifact.Type, err)
			}
		}

		if pipelineResult.OriginalSize.Width > 0 {
			data.ImageSize = &entity.Sizes{
				Width:  pipelineResult.OriginalSize.Width,
				Height: pipelineResult.OriginalSize.Height,
			}
		}
		if pipelineResult.ThumbnailSize.Width > 0 {
			data.ThumbSize = &entity.Sizes{
				Width:  pipelineResult.ThumbnailSize.Width,
				Height: pipelineResult.ThumbnailSize.Height,
			}
		}
	}

	if params.ComputeEmbedding && len(pipelineResult.Embedding) > 0 {
		data.EmbeddingList = []entity.EmbeddingItem{}
		for _, v := range pipelineResult.Embedding {
			if len(v.Data) >= 1 {
				data.EmbeddingList = append(data.EmbeddingList, v)
			}
		}
	}

	if len(pipelineResult.Tags) > 0 {
		data.Tags = helper.TransformSlice(
			pipelineResult.Tags,
			make([]string, len(pipelineResult.Tags)),
			func(tag entity.ElasticTag) string { return tag.Tag })
	}

	if err = r.metadataStorageService.Save(ctx, data); err != nil {
		return fmt.Errorf("save metadata failed: %w", err)
	}
	return nil
}

func (r *RecomputeServiceImpl) RecomputeOneById(ctx context.Context, accountId string, id string, params RecomputeParams) error {
	accountUuid, err := uuid.Parse(accountId)
	if err != nil {
		return fmt.Errorf("invalid account_id: %w", err)
	}
	idUuid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid media_id: %w", err)
	}
	data, err := r.metadataStorageService.GetById(ctx, accountUuid, idUuid)
	if err != nil {
		return fmt.Errorf("fetch metadata failed: %w", err)
	}
	if data == nil {
		return fmt.Errorf("media not found: %s", id)
	}
	return r.recomputeOne(ctx, data, params)
}

func NewRecomputeService(
	extractService MetadataExtractService,
	metadataStorageService storage2.MetadataStorageService,
	imageStorageService storage2.MediaStorageService,
	tmpDataService service.TmpDataService,
	jobStorage RecomputeJobStorage,
) RecomputeService {
	return &RecomputeServiceImpl{
		slogger:                slog.With("service", "RecomputeService"),
		extractService:         extractService,
		metadataStorageService: metadataStorageService,
		imageStorageService:    imageStorageService,
		tmpDataService:         tmpDataService,
		jobStorage:             jobStorage,
	}
}
