package api

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/youtube-service/entity"
	"github.com/weoses/memelo/youtube-service/service"
)

type YouTubeServiceApi struct {
	jobService service.VideoJobService
	log        *slog.Logger
}

func (api *YouTubeServiceApi) DownloadVideoSync(ctx context.Context, req *v1.DownloadVideoRequest) (*v1.DownloadVideoSyncResponse, error) {
	dlReq, err := api.parseRequest(req)
	if err != nil {
		return nil, err
	}

	result, err := api.jobService.RunSync(ctx, dlReq)
	if err != nil {
		api.log.ErrorContext(ctx, "DownloadVideoSync failed", "url", req.YoutubeUrl, "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return &v1.DownloadVideoSyncResponse{
		S3Path:   result.S3Path,
		MimeType: result.MimeType,
	}, nil
}

func (api *YouTubeServiceApi) DownloadVideoAsync(ctx context.Context, req *v1.DownloadVideoRequest) (*v1.DownloadVideoAsyncResponse, error) {
	dlReq, err := api.parseRequest(req)
	if err != nil {
		return nil, err
	}

	jobId, err := api.jobService.CreateJob(ctx, dlReq)
	if err != nil {
		api.log.ErrorContext(ctx, "DownloadVideoAsync failed to create job", "url", req.YoutubeUrl, "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return &v1.DownloadVideoAsyncResponse{JobId: jobId}, nil
}

func (api *YouTubeServiceApi) GetDownloadJobStatus(ctx context.Context, req *v1.GetDownloadJobStatusRequest) (*v1.GetDownloadJobStatusResponse, error) {
	if req.JobId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("job_id is required"))
	}

	job, err := api.jobService.GetJob(ctx, req.JobId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	job.Mu.RLock()
	defer job.Mu.RUnlock()

	resp := &v1.GetDownloadJobStatusResponse{
		JobId: job.JobId,
		State: toProtoJobState(job.State),
	}

	if job.Result != nil {
		resp.S3Path = &job.Result.S3Path
		resp.MimeType = &job.Result.MimeType
	}

	if job.Error != nil {
		errStr := job.Error.Error()
		resp.Error = &errStr
	}

	return resp, nil
}

func (api *YouTubeServiceApi) parseRequest(req *v1.DownloadVideoRequest) (entity.DownloadRequest, error) {
	if req.YoutubeUrl == "" {
		return entity.DownloadRequest{}, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("youtube_url is required"))
	}
	return entity.DownloadRequest{YoutubeURL: req.YoutubeUrl}, nil
}

func toProtoJobState(state entity.JobState) v1.DownloadJobState {
	switch state {
	case entity.JobStatePending:
		return v1.DownloadJobState_JOB_STATE_PENDING
	case entity.JobStateRunning:
		return v1.DownloadJobState_JOB_STATE_RUNNING
	case entity.JobStateDone:
		return v1.DownloadJobState_JOB_STATE_DONE
	case entity.JobStateFailed:
		return v1.DownloadJobState_JOB_STATE_FAILED
	default:
		return v1.DownloadJobState_JOB_STATE_UNSPECIFIED
	}
}

func NewYouTubeServiceApi(jobService service.VideoJobService) (*YouTubeServiceApi, error) {
	return &YouTubeServiceApi{
		jobService: jobService,
		log:        slog.With("service", "YouTubeServiceApi"),
	}, nil
}
