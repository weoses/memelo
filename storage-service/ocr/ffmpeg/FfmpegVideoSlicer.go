package ffmpeg

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/ocr"
)

type VideoSlicerImpl struct {
	slogger *slog.Logger
	cfg     *conf.FfmpegConfig
}

var _ ocr.VideoSlicer = (*VideoSlicerImpl)(nil)

func (f *VideoSlicerImpl) SliceVideo(ctx context.Context, video temp.Data, interval time.Duration) ([]temp.Data, error) {
	f.slogger.InfoContext(ctx, "SliceVideo: start", "interval", interval)

	dir, err := os.MkdirTemp("", "videoslicer-*")
	if err != nil {
		return nil, fmt.Errorf("SliceVideo: create temp dir: %w", err)
	}
	defer func() {
		if errRemoveAll := os.RemoveAll(dir); errRemoveAll != nil {
			f.slogger.WarnContext(ctx, "SliceVideo: remove temp dir failed", "error", errRemoveAll)
		}
	}()

	inputPath := filepath.Join(dir, "input.mp4")
	inputFile, err := os.OpenFile(inputPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("SliceVideo: create input file: %w", err)
	}

	videoReader, err := video.Reader()
	if err != nil {
		helper.QuietClose(inputFile, f.slogger)
		return nil, fmt.Errorf("SliceVideo: get reader: %w", err)
	}

	_, err = io.Copy(inputFile, videoReader)
	helper.QuietClose(inputFile, f.slogger)
	helper.QuietClose(videoReader, f.slogger)
	if err != nil {
		return nil, fmt.Errorf("SliceVideo: write input: %w", err)
	}

	outputPattern := filepath.Join(dir, "segment_%05d.mp4")
	intervalSec := strconv.FormatFloat(interval.Seconds(), 'f', -1, 64)
	cmd := buildCmd(ctx, f.cfg,
		"-i", inputPath,
		"-f", "segment",
		"-segment_time", intervalSec,
		"-reset_timestamps", "1",
		"-c", "copy",
		outputPattern,
	)
	f.slogger.InfoContext(ctx, "SliceVideo: running ffmpeg", "cmd", cmd.String())
	out, err := cmd.CombinedOutput()
	f.slogger.DebugContext(ctx, "SliceVideo: ffmpeg output", "output", string(out))
	if err != nil {
		return nil, fmt.Errorf("SliceVideo: ffmpeg failed: %w\n%s", err, out)
	}

	matches, err := filepath.Glob(filepath.Join(dir, "segment_*.mp4"))
	if err != nil {
		return nil, fmt.Errorf("SliceVideo: glob segments: %w", err)
	}
	sort.Strings(matches)

	slices := make([]temp.Data, 0, len(matches))
	for _, path := range matches {
		f.slogger.DebugContext(ctx, "SliceVideo: reading segment", "path", path)
		segFile, errOpen := os.Open(path)
		if errOpen != nil {
			closeAll(slices)
			return nil, fmt.Errorf("SliceVideo: open segment %s: %w", path, errOpen)
		}
		data, errData := temp.DataTemp(segFile)
		helper.QuietClose(segFile, f.slogger)
		if errData != nil {
			closeAll(slices)
			return nil, fmt.Errorf("SliceVideo: read segment %s: %w", path, errData)
		}
		slices = append(slices, data)
	}

	f.slogger.InfoContext(ctx, "SliceVideo: done", "segments", len(slices))
	return slices, nil
}

func closeAll(items []temp.Data) {
	for _, d := range items {
		_ = d.Close()
	}
}

func NewVideoSlicer(cfg *conf.FfmpegConfig) *VideoSlicerImpl {
	return &VideoSlicerImpl{
		slogger: slog.With("service", "VideoSlicer"),
		cfg:     cfg,
	}
}
