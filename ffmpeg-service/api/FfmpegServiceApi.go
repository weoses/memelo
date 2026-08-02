package api

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/weoses/memelo/ffmpeg-service/entity"
	"github.com/weoses/memelo/ffmpeg-service/service"
	v1 "github.com/weoses/memelo/gen/proto/v1"
)

type FfmpegServiceApi struct {
	jobService service.FfmpegJobService
	log        *slog.Logger
}

func (api *FfmpegServiceApi) SubmitJob(ctx context.Context, req *v1.SubmitFfmpegJobRequest) (*v1.SubmitFfmpegJobResponse, error) {
	jobReq, err := api.parseRequest(req)
	if err != nil {
		return nil, err
	}

	jobId, err := api.jobService.CreateJob(ctx, jobReq)
	if err != nil {
		api.log.ErrorContext(ctx, "SubmitJob failed", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return &v1.SubmitFfmpegJobResponse{JobId: jobId}, nil
}

func (api *FfmpegServiceApi) GetJobStatus(ctx context.Context, req *v1.GetFfmpegJobStatusRequest) (*v1.GetFfmpegJobStatusResponse, error) {
	if req.JobId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("job_id is required"))
	}

	job, err := api.jobService.GetJob(ctx, req.JobId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	job.Mu.RLock()
	defer job.Mu.RUnlock()

	resp := &v1.GetFfmpegJobStatusResponse{
		JobId: job.JobId,
		State: toProtoJobState(job.State),
	}

	if job.Result != nil {
		resp.OutputS3Path = &job.Result.OutputS3Path
		for _, slice := range job.Result.Slices {
			resp.Slices = append(resp.Slices, &v1.VideoSliceResult{
				S3Path:           slice.S3Path,
				StartTimeSeconds: int32(slice.StartTime),
				EndTimeSeconds:   int32(slice.EndTime),
			})
		}
	}

	if job.Error != nil {
		errStr := job.Error.Error()
		resp.Error = &errStr
	}

	return resp, nil
}

func (api *FfmpegServiceApi) parseRequest(req *v1.SubmitFfmpegJobRequest) (entity.JobRequest, error) {
	if req.InputS3Path == "" {
		return entity.JobRequest{}, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("input_s3_path is required"))
	}

	jobReq := entity.JobRequest{
		InputS3Path:      req.InputS3Path,
		RetentionSeconds: req.RetentionSeconds,
	}

	switch action := req.Action.(type) {
	case *v1.SubmitFfmpegJobRequest_ExtractThumbnail:
		jobReq.Action = entity.ActionExtractThumbnail
	case *v1.SubmitFfmpegJobRequest_ConvertToMp4:
		jobReq.Action = entity.ActionConvertToMp4
	case *v1.SubmitFfmpegJobRequest_SliceVideo:
		if action.SliceVideo.IntervalSeconds <= action.SliceVideo.OverlapSeconds {
			return entity.JobRequest{}, connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("interval_seconds (%d) must be greater than overlap_seconds (%d)",
					action.SliceVideo.IntervalSeconds, action.SliceVideo.OverlapSeconds))
		}
		jobReq.Action = entity.ActionSliceVideo
		jobReq.IntervalSeconds = action.SliceVideo.IntervalSeconds
		jobReq.OverlapSeconds = action.SliceVideo.OverlapSeconds
	default:
		return entity.JobRequest{}, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("action is required"))
	}

	return jobReq, nil
}

func toProtoJobState(state entity.JobState) v1.FfmpegJobState {
	switch state {
	case entity.JobStatePending:
		return v1.FfmpegJobState_FFMPEG_JOB_STATE_PENDING
	case entity.JobStateRunning:
		return v1.FfmpegJobState_FFMPEG_JOB_STATE_RUNNING
	case entity.JobStateDone:
		return v1.FfmpegJobState_FFMPEG_JOB_STATE_DONE
	case entity.JobStateFailed:
		return v1.FfmpegJobState_FFMPEG_JOB_STATE_FAILED
	default:
		return v1.FfmpegJobState_FFMPEG_JOB_STATE_UNSPECIFIED
	}
}

func NewFfmpegServiceApi(jobService service.FfmpegJobService) (*FfmpegServiceApi, error) {
	return &FfmpegServiceApi{
		jobService: jobService,
		log:        slog.With("service", "FfmpegServiceApi"),
	}, nil
}
