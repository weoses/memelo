package api

import (
	"context"
	"fmt"
	"log/slog"

	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"github.com/weoses/memelo/storage-service/service"
)

type RecomputeGrpcApiImpl struct {
	slogger          *slog.Logger
	recomputeService service.RecomputeService
}

func (r *RecomputeGrpcApiImpl) RecomputeById(ctx context.Context, request *v1.RecomputeOneRequest) (*v1.RecomputeOneResponse, error) {
	err := r.recomputeService.RecomputeOneById(ctx, request.AccountId, request.MediaId, service.RecomputeParams{
		ComputeExtractor:    request.ComputeExtractor,
		ComputeEmbedding:    request.ComputeEmbedding,
		CheckDuplicates:     request.CheckDuplicates,
		UpdateStorageItems:  request.UpdateStorageItems,
		IncludeManualEdited: request.IncludeManualEdited,
	})
	if err != nil {
		return nil, fmt.Errorf("recompute by id failed: %w", err)
	}
	return &v1.RecomputeOneResponse{Success: true}, nil
}

func (r *RecomputeGrpcApiImpl) StartRecomputeJob(ctx context.Context, req *v1.RecomputeRequest) (*v1.RecomputeJob, error) {
	var query map[string]interface{}
	if req.Query != nil {
		query = req.Query.AsMap()
	}

	jobId, err := r.recomputeService.StartRecompute(ctx, service.RecomputeParams{
		Query:               query,
		ComputeExtractor:    req.ComputeExtractor,
		ComputeEmbedding:    req.ComputeEmbedding,
		CheckDuplicates:     req.CheckDuplicates,
		UpdateStorageItems:  req.UpdateStorageItems,
		IncludeManualEdited: req.IncludeManualEdited,
	})
	if err != nil {
		return nil, fmt.Errorf("start recompute failed: %w", err)
	}
	return &v1.RecomputeJob{JobId: jobId}, nil
}

func (r *RecomputeGrpcApiImpl) GetRecomputeJobStatus(ctx context.Context, req *v1.RecomputeJob) (*v1.RecomputeJobStatus, error) {
	job, err := r.recomputeService.GetJobStatus(ctx, req.JobId)
	if err != nil {
		return nil, fmt.Errorf("get recompute status failed: %w", err)
	}

	job.Mu.RLock()
	defer job.Mu.RUnlock()

	errors := make([]*v1.RecomputeError, len(job.Errors))
	for i, e := range job.Errors {
		errors[i] = &v1.RecomputeError{
			ObjectId:  e.ObjectId,
			ErrorText: e.ErrorText,
		}
	}

	return &v1.RecomputeJobStatus{
		JobId:     job.JobId,
		State:     stateToProto(job.State),
		Processed: job.Processed,
		Total:     job.Total,
		Errors:    errors,
		LastId:    job.LastId,
	}, nil
}

func stateToProto(s service.RecomputeState) v1.RecomputeJobStatus_State {
	switch s {
	case service.RecomputeStatePending:
		return v1.RecomputeJobStatus_STATE_PENDING
	case service.RecomputeStateRunning:
		return v1.RecomputeJobStatus_STATE_RUNNING
	case service.RecomputeStateDone:
		return v1.RecomputeJobStatus_STATE_DONE
	case service.RecomputeStateFailed:
		return v1.RecomputeJobStatus_STATE_FAILED
	default:
		return v1.RecomputeJobStatus_STATE_UNSPECIFIED
	}
}

func NewRecomputeGrpcApi(svc service.RecomputeService) v1connect.RecomputeServiceHandler {
	return &RecomputeGrpcApiImpl{
		slogger:          slog.With("service", "RecomputeGrpcApi"),
		recomputeService: svc,
	}
}
