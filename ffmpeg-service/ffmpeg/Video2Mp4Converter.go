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

func ConvertToMp4(ctx context.Context, cfg *conf.FfmpegConfig, video temp.Data, slogger *slog.Logger) (temp.Data, error) {
	slogger.InfoContext(ctx, "ConvertToMp4: start")

	dir, err := os.MkdirTemp("", "video2mp4-*")
	if err != nil {
		return nil, fmt.Errorf("ConvertToMp4: create temp dir: %w", err)
	}
	defer func(path string) {
		errRemoveAll := os.RemoveAll(path)
		if errRemoveAll != nil {
			slogger.WarnContext(ctx, "ConvertToMp4: remove temp dir failed: ", "error", errRemoveAll)
		}
	}(dir)

	ffmpegInputPath := filepath.Join(dir, "input")
	ffmpegInputFile, err := os.OpenFile(ffmpegInputPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("ConvertToMp4: create input file: %w", err)
	}

	videoInputReader, err := video.Reader()
	if err != nil {
		helper.QuietClose(ffmpegInputFile, slogger)
		return nil, fmt.Errorf("ConvertToMp4: get videoInputReader: %w", err)
	}

	_, err = io.Copy(ffmpegInputFile, videoInputReader)
	helper.QuietClose(ffmpegInputFile, slogger)
	helper.QuietClose(videoInputReader, slogger)
	if err != nil {
		return nil, fmt.Errorf("ConvertToMp4: write input: %w", err)
	}

	outputPath := filepath.Join(dir, "output.mp4")
	cmd := buildCmd(ctx, cfg,
		"-i", ffmpegInputPath,
		"-c:v", "libx264",
		"-c:a", "aac",
		"-movflags", "+faststart",
		outputPath,
	)
	slogger.InfoContext(ctx, "ConvertToMp4: running ffmpeg", "cmd", cmd.String())
	out, err := cmd.CombinedOutput()
	slogger.DebugContext(ctx, "ConvertToMp4: ffmpeg output", "output", string(out))
	if err != nil {
		return nil, fmt.Errorf("ConvertToMp4: ffmpeg failed: %w\n%s", err, out)
	}

	outputFile, err := os.Open(outputPath)
	if err != nil {
		return nil, fmt.Errorf("ConvertToMp4: open output: %w", err)
	}
	defer helper.QuietClose(outputFile, slogger)

	data, err := temp.DataTemp(outputFile)
	if err != nil {
		return nil, fmt.Errorf("ConvertToMp4: read output: %w", err)
	}

	slogger.InfoContext(ctx, "ConvertToMp4: done")
	return data, nil
}
