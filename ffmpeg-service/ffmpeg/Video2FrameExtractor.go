package ffmpeg

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/ffmpeg-service/conf"
)

func ExtractOneFrame(ctx context.Context, cfg *conf.FfmpegConfig, video temp.Data, slogger *slog.Logger) (temp.Data, error) {
	slogger.InfoContext(ctx, "ExtractOneFrame: start")

	dir, err := os.MkdirTemp("", "video2thumb-*")
	if err != nil {
		return nil, fmt.Errorf("ExtractOneFrame: create temp dir: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			slogger.WarnContext(ctx, "ExtractOneFrame: remove temp dir failed", "error", err)
		}
	}()

	ffmpegInputPath := filepath.Join(dir, "input.mp4")
	ffmpegInputFile, err := os.OpenFile(ffmpegInputPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("ExtractOneFrame: create input file: %w", err)
	}

	videoInputReader, err := video.Reader(ctx)
	if err != nil {
		helper.QuietClose(ffmpegInputFile, slogger)
		return nil, fmt.Errorf("ExtractOneFrame: get reader: %w", err)
	}

	_, err = io.Copy(ffmpegInputFile, videoInputReader)
	helper.QuietClose(videoInputReader, slogger)
	helper.QuietClose(ffmpegInputFile, slogger)
	if err != nil {
		return nil, fmt.Errorf("ExtractOneFrame: write input: %w", err)
	}

	outputPath := filepath.Join(dir, "thumb.jpg")
	cmd := buildCmd(ctx, cfg,
		"-i", ffmpegInputPath,
		"-vframes", "1",
		"-f", "image2",
		outputPath,
	)
	slogger.InfoContext(ctx, "ExtractOneFrame: running ffmpeg", "cmd", cmd.String())
	out, err := cmd.CombinedOutput()
	if out != nil {
		slogger.DebugContext(ctx, "ExtractOneFrame: ffmpeg output", "output", string(out))
	}
	if err != nil {
		return nil, fmt.Errorf("ExtractOneFrame: ffmpeg failed: %w\n%s", err, out)
	}

	return processExtractedFrame(outputPath, slogger)
}

func processExtractedFrame(path string, slogger *slog.Logger) (temp.Data, error) {
	frameFile, err := os.Open(path)
	defer helper.QuietClose(frameFile, slogger)

	if err != nil {
		return nil, fmt.Errorf("ExtractOneFrame: open frame %s: %w", path, err)
	}
	data, err := temp.DataTemp(frameFile)

	if err != nil {
		return nil, fmt.Errorf("ExtractOneFrame: read frame %s: %w", path, err)
	}
	return data, nil
}
