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
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/ocr"
)

type Video2AudioExtractorImpl struct {
	slogger *slog.Logger
	cfg     *conf.FfmpegConfig
}

var _ ocr.Video2AudioExtractor = (*Video2AudioExtractorImpl)(nil)

func (f *Video2AudioExtractorImpl) ExtractAudio(ctx context.Context, video temp.Data) (temp.Data, error) {
	f.slogger.InfoContext(ctx, "ExtractAudio: start")

	dir, err := os.MkdirTemp("", "video2audio-*")
	if err != nil {
		return nil, fmt.Errorf("ExtractAudio: create temp dir: %w", err)
	}
	defer func(path string) {
		errRemoveAll := os.RemoveAll(path)
		if errRemoveAll != nil {
			f.slogger.WarnContext(ctx, "ExtractAudio: remove temp dir failed: ", "error", errRemoveAll)
		}
	}(dir)

	ffmpegInputPath := filepath.Join(dir, "input.mp4")
	ffmpegInputFile, err := os.OpenFile(ffmpegInputPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("ExtractAudio: create input file: %w", err)
	}

	videoInputReader, err := video.Reader()
	if err != nil {
		helper.QuietClose(ffmpegInputFile, f.slogger)
		return nil, fmt.Errorf("ExtractAudio: get videoInputReader: %w", err)
	}

	_, err = io.Copy(ffmpegInputFile, videoInputReader)
	helper.QuietClose(ffmpegInputFile, f.slogger)
	helper.QuietClose(videoInputReader, f.slogger)
	if err != nil {
		return nil, fmt.Errorf("ExtractAudio: write input: %w", err)
	}

	outputPath := filepath.Join(dir, "audio.wav")
	cmd := buildCmd(ctx, f.cfg,
		"-i", ffmpegInputPath,
		"-vn",
		"-acodec", "pcm_s16le",
		"-ar", "44100",
		"-ac", "1",
		outputPath,
	)
	f.slogger.InfoContext(ctx, "ExtractAudio: running ffmpeg", "cmd", cmd.String())
	out, err := cmd.CombinedOutput()
	f.slogger.DebugContext(ctx, "ExtractAudio: ffmpeg output", "output", string(out))
	if err != nil {
		return nil, fmt.Errorf("ExtractAudio: ffmpeg failed: %w\n%s", err, out)
	}

	outputFile, err := os.Open(outputPath)
	if err != nil {
		return nil, fmt.Errorf("ExtractAudio: open output: %w", err)
	}
	defer helper.QuietClose(outputFile, f.slogger)

	data, err := temp.DataTemp(outputFile)
	if err != nil {
		return nil, fmt.Errorf("ExtractAudio: read output: %w", err)
	}

	f.slogger.InfoContext(ctx, "ExtractAudio: done")
	return data, nil
}

func NewVideo2AudioExtractor(cfg *conf.FfmpegConfig) ocr.Video2AudioExtractor {
	return &Video2AudioExtractorImpl{
		slogger: slog.With("service", "Video2AudioExtractor"),
		cfg:     cfg,
	}
}
