package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/weoses/memelo/common/temp"
	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"github.com/weoses/memelo/youtube-service/conf"
	"github.com/weoses/memelo/youtube-service/entity"
)

type StorageConnector interface {
	CreateMeme(ctx context.Context, data temp.S3BackedData, accountId uuid.UUID) (*entity.VideoCreateResult, error)
}

type storageConnectorImpl struct {
	cl  v1connect.SearchServiceClient
	log *slog.Logger
}

func (s *storageConnectorImpl) CreateMeme(ctx context.Context, data temp.S3BackedData, accountId uuid.UUID) (*entity.VideoCreateResult, error) {
	s3path, err := data.GetS3Path(ctx)
	if err != nil {
		return nil, fmt.Errorf("StorageConnector: failed to get s3 path: %w", err)
	}

	response, err := s.cl.CreateMeme(ctx, &v1.CreateMemeRequest{
		AccountId: accountId.String(),
		Video:     &v1.MediaDataDto{S3Path: &s3path},
	})
	if err != nil {
		return nil, fmt.Errorf("StorageConnector: CreateMeme RPC failed: %w", err)
	}

	memeId, err := uuid.Parse(response.Result.GetId())
	if err != nil {
		return nil, fmt.Errorf("StorageConnector: failed to parse meme id %q: %w", response.Result.GetId(), err)
	}

	return &entity.VideoCreateResult{
		Id:              memeId,
		DuplicateStatus: response.Status.String(),
		Caption:         response.Result.GetCaption(),
		Tags:            response.Result.GetTags(),
		OcrResult:       response.Result.GetOcrResult(),
	}, nil
}

func NewStorageConnector(cfg *conf.Config) (StorageConnector, error) {
	cl := v1connect.NewSearchServiceClient(http.DefaultClient, cfg.StorageService.Uri)
	return &storageConnectorImpl{
		cl:  cl,
		log: slog.With("service", "StorageConnector"),
	}, nil
}
