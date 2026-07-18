package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"github.com/weoses/memelo/telegram-service/conf"
	"github.com/weoses/memelo/telegram-service/entity"
)

type YouTubeConnector interface {
	DownloadVideoAsync(ctx context.Context, url string) (string, error)
	GetDownloadJobStatus(ctx context.Context, jobId string) (*entity.YouTubeJobStatus, error)
}

type YouTubeConnectorImpl struct {
	cl  v1connect.YouTubeServiceClient
	log *slog.Logger
}

func (y *YouTubeConnectorImpl) DownloadVideoAsync(ctx context.Context, url string) (string, error) {
	response, err := y.cl.DownloadVideoAsync(ctx, &v1.DownloadVideoRequest{
		YoutubeUrl: url,
	})
	if err != nil {
		return "", fmt.Errorf("YouTubeConnector: DownloadVideoAsync failed: %w", err)
	}
	return response.GetJobId(), nil
}

func (y *YouTubeConnectorImpl) GetDownloadJobStatus(ctx context.Context, jobId string) (*entity.YouTubeJobStatus, error) {
	response, err := y.cl.GetDownloadJobStatus(ctx, &v1.GetDownloadJobStatusRequest{
		JobId: jobId,
	})
	if err != nil {
		return nil, fmt.Errorf("YouTubeConnector: GetDownloadJobStatus failed: %w", err)
	}

	status := &entity.YouTubeJobStatus{
		State:    toJobStateString(response.GetState()),
		S3Path:   response.GetS3Path(),
		MimeType: response.GetMimeType(),
	}
	if response.Error != nil {
		status.Error = *response.Error
	}
	return status, nil
}

func toJobStateString(state v1.DownloadJobState) string {
	switch state {
	case v1.DownloadJobState_JOB_STATE_PENDING:
		return "pending"
	case v1.DownloadJobState_JOB_STATE_RUNNING:
		return "running"
	case v1.DownloadJobState_JOB_STATE_DONE:
		return "done"
	case v1.DownloadJobState_JOB_STATE_FAILED:
		return "failed"
	default:
		return "unknown"
	}
}

func NewYouTubeConnector(cfg *conf.Config) (YouTubeConnector, error) {
	cl := v1connect.NewYouTubeServiceClient(http.DefaultClient, cfg.YoutubeService.Uri)
	return &YouTubeConnectorImpl{
		cl:  cl,
		log: slog.With("service", "YouTubeConnector"),
	}, nil
}
