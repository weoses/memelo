package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"github.com/weoses/memelo/telegram-service/conf"
	"github.com/weoses/memelo/telegram-service/entity"
)

type YouTubeConnector interface {
	DownloadVideoSync(ctx context.Context, url string, accountId uuid.UUID) (*entity.MemeCreateResult, error)
}

type YouTubeConnectorImpl struct {
	cl  v1connect.YouTubeServiceClient
	log *slog.Logger
}

func (y *YouTubeConnectorImpl) DownloadVideoSync(ctx context.Context, url string, accountId uuid.UUID) (*entity.MemeCreateResult, error) {
	response, err := y.cl.DownloadVideoSync(ctx, &v1.DownloadVideoRequest{
		YoutubeUrl: url,
		AccountId:  accountId.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("YouTubeConnector: DownloadVideoSync failed: %w", err)
	}

	memeId, err := uuid.Parse(response.GetResult().GetId())
	if err != nil {
		return nil, fmt.Errorf("YouTubeConnector: failed to parse meme id: %s: %w", response.GetResult().GetId(), err)
	}
	return &entity.MemeCreateResult{
		Id:              memeId,
		Text:            response.GetResult().GetOcrResult(),
		Tags:            response.GetResult().GetTags(),
		Caption:         response.GetResult().GetCaption(),
		DuplicateStatus: response.GetStatus().String(),
	}, nil
}

func NewYouTubeConnector(cfg *conf.Config) (YouTubeConnector, error) {
	cl := v1connect.NewYouTubeServiceClient(http.DefaultClient, cfg.YoutubeService.Uri)
	return &YouTubeConnectorImpl{
		cl:  cl,
		log: slog.With("service", "YouTubeConnector"),
	}, nil
}
