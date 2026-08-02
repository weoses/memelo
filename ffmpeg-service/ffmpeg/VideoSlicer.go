package ffmpeg

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/ffmpeg-service/conf"
)

type VideoSlice struct {
	Data      temp.Data
	StartTime int // seconds
	EndTime   int // seconds
}

func SliceVideoWithOverlap(
	ctx context.Context,
	cfg *conf.FfmpegConfig,
	video temp.Data,
	interval time.Duration,
	overlap time.Duration,
	slogger *slog.Logger,
) ([]VideoSlice, error) {

	slogger.InfoContext(ctx, "SliceVideoWithOverlap: start", "interval", interval, "overlap", overlap)

	if interval <= overlap {
		return nil, fmt.Errorf("SliceVideoWithOverlap: interval (%s) must be greater than overlap (%s)", interval, overlap)
	}

	dir, err := os.MkdirTemp("", "videoslicer-overlap-*")
	if err != nil {
		return nil, fmt.Errorf("SliceVideoWithOverlap: create temp dir: %w", err)
	}
	defer func() {
		if errRemoveAll := os.RemoveAll(dir); errRemoveAll != nil {
			slogger.WarnContext(ctx, "SliceVideoWithOverlap: remove temp dir failed", "error", errRemoveAll)
		}
	}()

	inputPath := filepath.Join(dir, "input.mp4")
	inputFile, err := os.OpenFile(inputPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("SliceVideoWithOverlap: create input file: %w", err)
	}

	videoReader, err := video.Reader()
	if err != nil {
		helper.QuietClose(inputFile, slogger)
		return nil, fmt.Errorf("SliceVideoWithOverlap: get reader: %w", err)
	}

	_, err = io.Copy(inputFile, videoReader)
	helper.QuietClose(inputFile, slogger)
	helper.QuietClose(videoReader, slogger)
	if err != nil {
		return nil, fmt.Errorf("SliceVideoWithOverlap: write input: %w", err)
	}

	totalDuration, err := getVideoDuration(ctx, cfg, inputPath)
	if err != nil {
		return nil, fmt.Errorf("SliceVideoWithOverlap: get duration: %w", err)
	}
	slogger.InfoContext(ctx, "SliceVideoWithOverlap: video duration", "duration", totalDuration)

	step := interval - overlap
	slices := make([]VideoSlice, 0)
	for start := time.Duration(0); start < totalDuration; start += step {
		end := start + interval
		if end > totalDuration {
			end = totalDuration
		}

		if (end-start) < overlap && start > 0 {
			break
		}

		segmentPath := filepath.Join(dir, fmt.Sprintf("segment_%05d.mp4", len(slices)))
		startSec := strconv.FormatFloat(start.Seconds(), 'f', -1, 64)
		durationSec := strconv.FormatFloat(interval.Seconds(), 'f', -1, 64)

		cmd := buildCmd(ctx, cfg,
			"-ss", startSec,
			"-i", inputPath,
			"-t", durationSec,
			"-c", "copy",
			segmentPath,
		)
		slogger.InfoContext(ctx, "SliceVideoWithOverlap: running ffmpeg", "cmd", cmd.String(), "start", startSec, "sliceStart", start, "duration", durationSec)
		out, errCmd := cmd.CombinedOutput()
		slogger.DebugContext(ctx, "SliceVideoWithOverlap: ffmpeg output", "output", string(out))
		if errCmd != nil {
			closeAll(slices)
			return nil, fmt.Errorf("SliceVideoWithOverlap: ffmpeg failed: %w\n%s", errCmd, out)
		}

		segFile, errOpen := os.Open(segmentPath)
		if errOpen != nil {
			closeAll(slices)
			return nil, fmt.Errorf("SliceVideoWithOverlap: open segment: %w", errOpen)
		}
		data, errData := temp.DataTemp(segFile)
		helper.QuietClose(segFile, slogger)
		if errData != nil {
			closeAll(slices)
			return nil, fmt.Errorf("SliceVideoWithOverlap: read segment: %w", errData)
		}
		slices = append(slices, VideoSlice{
			Data:      data,
			StartTime: int(start.Seconds()),
			EndTime:   int(end.Seconds()),
		})
	}

	slogger.InfoContext(ctx, "SliceVideoWithOverlap: done", "segments", len(slices))
	return slices, nil
}

func getVideoDuration(ctx context.Context, cfg *conf.FfmpegConfig, inputPath string) (time.Duration, error) {
	cmd := exec.CommandContext(ctx, cfg.FfprobeBinary,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		inputPath,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	durStr := strings.TrimSpace(string(out))
	durSec, err := strconv.ParseFloat(durStr, 64)
	if err != nil {
		return 0, fmt.Errorf("parse duration %q: %w", durStr, err)
	}
	return time.Duration(durSec * float64(time.Second)), nil
}

func closeAll(items []VideoSlice) {
	for _, d := range items {
		_ = d.Data.Close()
	}
}
