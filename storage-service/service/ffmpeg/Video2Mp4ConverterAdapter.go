package ffmpeg

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/weoses/memelo/common/helper"
	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/common/temp"
	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/ocr"
)

type video2Mp4ConverterAdapter struct {
	cl             v1connect.FfmpegServiceClient
	tmpDataService commonservice.TmpDataService
	pollInterval   time.Duration
	pollMaxWait    time.Duration
	log            *slog.Logger
}

var _ ocr.Video2Mp4Converter = (*video2Mp4ConverterAdapter)(nil)

func (a *video2Mp4ConverterAdapter) ConvertToMp4(ctx context.Context, video temp.Data) (temp.Data, error) {
	input, err := resolveInput(ctx, a.tmpDataService, video)
	if err != nil {
		return nil, fmt.Errorf("Video2Mp4ConverterAdapter: %w", err)
	}
	if input.Owned {
		defer helper.QuietCloseCtx(ctx, input.Created, a.log)
	}

	submitResp, err := a.cl.SubmitJob(ctx, &v1.SubmitFfmpegJobRequest{
		InputS3Path: input.S3Path,
		Action:      &v1.SubmitFfmpegJobRequest_ConvertToMp4{ConvertToMp4: &v1.ConvertToMp4Action{}},
	})
	if err != nil {
		return nil, fmt.Errorf("Video2Mp4ConverterAdapter: submit job: %w", err)
	}

	status, err := pollJob(ctx, a.cl, submitResp.JobId, a.pollInterval, a.pollMaxWait)
	if err != nil {
		return nil, fmt.Errorf("Video2Mp4ConverterAdapter: %w", err)
	}

	result, err := a.tmpDataService.WrapInternalS3Path(ctx, status.GetOutputS3Path())
	if err != nil {
		return nil, fmt.Errorf("Video2Mp4ConverterAdapter: wrap result: %w", err)
	}
	return result, nil
}

func NewVideo2Mp4ConverterAdapter(cfg *conf.FfmpegServiceConfig, tmpDataService commonservice.TmpDataService) (ocr.Video2Mp4Converter, error) {
	cl, err := newClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("NewVideo2Mp4ConverterAdapter: %w", err)
	}
	return &video2Mp4ConverterAdapter{
		cl:             cl,
		tmpDataService: tmpDataService,
		pollInterval:   time.Duration(cfg.PollIntervalMs) * time.Millisecond,
		pollMaxWait:    time.Duration(cfg.PollMaxWaitSec) * time.Second,
		log:            slog.With("service", "Video2Mp4ConverterAdapter"),
	}, nil
}
