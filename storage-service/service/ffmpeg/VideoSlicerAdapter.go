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

type videoSlicerAdapter struct {
	cl             v1connect.FfmpegServiceClient
	tmpDataService commonservice.TmpDataService
	pollInterval   time.Duration
	pollMaxWait    time.Duration
	log            *slog.Logger
}

var _ ocr.VideoSlicer = (*videoSlicerAdapter)(nil)

func (a *videoSlicerAdapter) SliceVideoWithOverlap(
	ctx context.Context,
	video temp.Data,
	interval time.Duration,
	overlap time.Duration,
) ([]ocr.VideoSlice, error) {
	input, err := resolveInput(ctx, a.tmpDataService, video)
	if err != nil {
		return nil, fmt.Errorf("VideoSlicerAdapter: %w", err)
	}
	if input.Owned {
		defer helper.QuietClose(input.Created, a.log)
	}

	submitResp, err := a.cl.SubmitJob(ctx, &v1.SubmitFfmpegJobRequest{
		InputS3Path: input.S3Path,
		Action: &v1.SubmitFfmpegJobRequest_SliceVideo{
			SliceVideo: &v1.SliceVideoAction{
				IntervalSeconds: int32(interval.Seconds()),
				OverlapSeconds:  int32(overlap.Seconds()),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("VideoSlicerAdapter: submit job: %w", err)
	}

	status, err := pollJob(ctx, a.cl, submitResp.JobId, a.pollInterval, a.pollMaxWait)
	if err != nil {
		return nil, fmt.Errorf("VideoSlicerAdapter: %w", err)
	}

	slices := make([]ocr.VideoSlice, 0, len(status.GetSlices()))
	for i, s := range status.GetSlices() {
		data, err := a.tmpDataService.WrapInternalS3Path(ctx, s.GetS3Path())
		if err != nil {
			for _, wrapped := range slices[:i] {
				helper.QuietClose(wrapped.Data, a.log)
			}
			return nil, fmt.Errorf("VideoSlicerAdapter: wrap slice %d: %w", i, err)
		}
		slices = append(slices, ocr.VideoSlice{
			Data:      data,
			StartTime: int(s.GetStartTimeSeconds()),
			EndTime:   int(s.GetEndTimeSeconds()),
		})
	}
	return slices, nil
}

func NewVideoSlicerAdapter(cfg *conf.FfmpegServiceConfig, tmpDataService commonservice.TmpDataService) (ocr.VideoSlicer, error) {
	cl, err := newClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("NewVideoSlicerAdapter: %w", err)
	}
	return &videoSlicerAdapter{
		cl:             cl,
		tmpDataService: tmpDataService,
		pollInterval:   time.Duration(cfg.PollIntervalMs) * time.Millisecond,
		pollMaxWait:    time.Duration(cfg.PollMaxWaitSec) * time.Second,
		log:            slog.With("service", "VideoSlicerAdapter"),
	}, nil
}
