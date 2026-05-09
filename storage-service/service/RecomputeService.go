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

type RecomputeOptions struct {
	ComputeExtractor   bool
	ComputeEmbedding   bool
	UpdateStorageItems bool
	CheckDuplicates    bool

	IncludeManualEdited bool
}

type RecomputeService interface {
	StartRecompute(ctx context.Context, query map[string]interface{}, params RecomputeOptions) (string, error)
	StartRecomputeById(ctx context.Context, accountId string, id string, params RecomputeOptions) (string, error)

	GetJobStatus(ctx context.Context, jobId string) (*RecomputeJobState, error)
	WaitForJob(ctx context.Context, jobId string) (*RecomputeJobState, error)
}

type RecomputeServiceImpl struct {
	slogger                *slog.Logger
	extractService         MetadataExtractService
	metadataStorageService storage2.MetadataStorageService
	imageStorageService    storage2.MediaStorageService
	pipelineSaveService    PipelineSaveService
	tmpDataService         service.TmpDataService
	jobStorage             RecomputeJobStorage
}

func (r *RecomputeServiceImpl) StartRecompute(ctx context.Context, query map[string]interface{}, params RecomputeOptions) (string, error) {

	if query == nil {
		query = map[string]interface{}{"match_all": map[string]interface{}{}}
	}

	jobId := uuid.New().String()
	job := r.jobStorage.Create(jobId)

	go r.runJob(context.WithoutCancel(ctx), query, job, params)

	return jobId, nil
}

func (r *RecomputeServiceImpl) StartRecomputeById(ctx context.Context, accountId string, id string, params RecomputeOptions) (string, error) {
	query := map[string]interface{}{
		"bool": map[string]interface{}{
			"must": map[string]interface{}{
				"ids": map[string]interface{}{
					"values": []string{id},
				},
			},
			"filter": []interface{}{
				map[string]interface{}{
					"term": map[string]interface{}{
						"AccountId": accountId,
					},
				},
			},
		},
	}

	jobId := uuid.New().String()
	job := r.jobStorage.Create(jobId)

	go r.runJob(context.WithoutCancel(ctx), query, job, params)

	return jobId, nil
}

func (r *RecomputeServiceImpl) GetJobStatus(_ context.Context, jobId string) (*RecomputeJobState, error) {
	job, ok := r.jobStorage.Get(jobId)
	if !ok {
		return nil, fmt.Errorf("job not found: %s", jobId)
	}
	return job, nil
}

func (r *RecomputeServiceImpl) WaitForJob(ctx context.Context, jobId string) (*RecomputeJobState, error) {
	job, ok := r.jobStorage.Get(jobId)
	if !ok {
		return nil, fmt.Errorf("job not found: %s", jobId)
	}

	for {
		job.Mu.Lock()
		if job.State == RecomputeStateDone || job.State == RecomputeStateFailed {
			break
		}
		job.StateCondition.Wait()
		job.Mu.Unlock()
	}

	return job, nil
}

func (r *RecomputeServiceImpl) runJob(ctx context.Context, query map[string]interface{}, job *RecomputeJobState, params RecomputeOptions) {
	job.Mu.Lock()
	job.State = RecomputeStateRunning
	job.StateCondition.Broadcast()
	job.Mu.Unlock()

	rawQuery := query

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
			job.StateCondition.Broadcast()
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

				err := r.recomputeOne(ctx, m, params)
				if err != nil {
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
				}
				r.slogger.InfoContext(ctx, "recompute: completed media object",
					"jobId", job.JobId,
					"imageId", m.ImageId)

				job.Mu.Lock()
				job.ProcessedIds = append(job.ProcessedIds, m.ImageId.String())
				job.Processed++
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
	job.StateCondition.Broadcast()
	job.Mu.Unlock()

	r.slogger.InfoContext(ctx, "recompute: job done", "jobId", job.JobId, "processed", job.Processed)
}

func (r *RecomputeServiceImpl) recomputeOne(ctx context.Context, data *entity.ElasticImageMetaData, params RecomputeOptions) error {
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

	data.Edited = false

	return r.pipelineSaveService.Save(ctx, data, pipelineResult, PipelineSaveConfig{
		SaveHash:      true,
		SaveExtractor: params.ComputeExtractor,
		SaveArtifacts: params.UpdateStorageItems,
		SaveEmbedding: params.ComputeEmbedding,
	})
}

func (r *RecomputeServiceImpl) RecomputeOneById(ctx context.Context, accountId string, id string, params RecomputeOptions) error {
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
	pipelineSaveService PipelineSaveService,
	tmpDataService service.TmpDataService,
	jobStorage RecomputeJobStorage,
) RecomputeService {
	return &RecomputeServiceImpl{
		slogger:                slog.With("service", "RecomputeService"),
		extractService:         extractService,
		metadataStorageService: metadataStorageService,
		imageStorageService:    imageStorageService,
		pipelineSaveService:    pipelineSaveService,
		tmpDataService:         tmpDataService,
		jobStorage:             jobStorage,
	}
}
