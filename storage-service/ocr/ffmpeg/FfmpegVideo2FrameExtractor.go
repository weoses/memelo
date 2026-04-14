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

type Video2FrameExtractorImpl struct {
	slogger *slog.Logger
	cfg     *conf.FfmpegConfig
}

var _ ocr.Video2FrameExtractor = (*Video2FrameExtractorImpl)(nil)

func (f *Video2FrameExtractorImpl) ExtractFrames(ctx context.Context, video temp.Data) ([]temp.Data, error) {
	f.slogger.InfoContext(ctx, "ExtractFrames: start")

	dir, err := os.MkdirTemp("", "video2frames-*")
	if err != nil {
		return nil, fmt.Errorf("ExtractFrames: create temp dir: %w", err)
	}
	defer func(path string) {
		errRemoveAll := os.RemoveAll(path)
		if errRemoveAll != nil {
			f.slogger.WarnContext(ctx, "ExtractFrames: remove temp dir failed: ", "error", errRemoveAll)
		}
	}(dir)

	ffmpegInputPath := filepath.Join(dir, "input.mp4")
	ffmpegInputFile, err := os.OpenFile(ffmpegInputPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("ExtractFrames: create input file: %w", err)
	}

	videoInputReader, err := video.Reader()
	if err != nil {
		helper.QuietClose(ffmpegInputFile, f.slogger)
		return nil, fmt.Errorf("ExtractFrames: get videoInputReader: %w", err)
	}

	_, err = io.Copy(ffmpegInputFile, videoInputReader)
	helper.QuietClose(videoInputReader, f.slogger)
	helper.QuietClose(ffmpegInputFile, f.slogger)
	if err != nil {
		return nil, fmt.Errorf("ExtractFrames: write input: %w", err)
	}

	outputPattern := filepath.Join(dir, "frame%06d.jpg")
	cmd := buildCmd(ctx, f.cfg,
		"-i", ffmpegInputPath,
		"-vf", "fps=1",
		"-f", "image2",
		outputPattern,
	)
	f.slogger.InfoContext(ctx, "ExtractFrames: running ffmpeg", "cmd", cmd.String())
	out, err := cmd.CombinedOutput()
	if out != nil {
		f.slogger.DebugContext(ctx, "ExtractFrames: ffmpeg output", "output", string(out))
	}

	if err != nil {
		return nil, fmt.Errorf("ExtractFrames: ffmpeg failed: %w\n%s", err, out)
	}

	matches, err := filepath.Glob(filepath.Join(dir, "frame*.jpg"))
	if err != nil {
		return nil, fmt.Errorf("ExtractFrames: glob frames: %w", err)
	}
	f.slogger.InfoContext(ctx, "ExtractFrames: frames produced", "count", len(matches), "files", matches)

	frames := make([]temp.Data, len(matches))
	for i, path := range matches {
		frameData, err := f.processExtractedFrame(path)
		if err != nil {
			helper.QuietCloseAll(frames, f.slogger)
			return nil, err
		}
		frames[i] = frameData
	}

	f.slogger.InfoContext(ctx, "ExtractFrames: done", "frames", len(frames))
	return frames, nil
}

func (f *Video2FrameExtractorImpl) ExtractThumbnail(ctx context.Context, video temp.Data) (temp.Data, error) {
	f.slogger.InfoContext(ctx, "ExtractThumbnail: start")

	dir, err := os.MkdirTemp("", "video2thumb-*")
	if err != nil {
		return nil, fmt.Errorf("ExtractThumbnail: create temp dir: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			f.slogger.WarnContext(ctx, "ExtractThumbnail: remove temp dir failed", "error", err)
		}
	}()

	ffmpegInputPath := filepath.Join(dir, "input.mp4")
	ffmpegInputFile, err := os.OpenFile(ffmpegInputPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("ExtractThumbnail: create input file: %w", err)
	}

	videoInputReader, err := video.Reader()
	if err != nil {
		helper.QuietClose(ffmpegInputFile, f.slogger)
		return nil, fmt.Errorf("ExtractThumbnail: get reader: %w", err)
	}

	_, err = io.Copy(ffmpegInputFile, videoInputReader)
	helper.QuietClose(videoInputReader, f.slogger)
	helper.QuietClose(ffmpegInputFile, f.slogger)
	if err != nil {
		return nil, fmt.Errorf("ExtractThumbnail: write input: %w", err)
	}

	outputPath := filepath.Join(dir, "thumb.jpg")
	cmd := buildCmd(ctx, f.cfg,
		"-i", ffmpegInputPath,
		"-vframes", "1",
		"-f", "image2",
		outputPath,
	)
	f.slogger.InfoContext(ctx, "ExtractThumbnail: running ffmpeg", "cmd", cmd.String())
	out, err := cmd.CombinedOutput()
	if out != nil {
		f.slogger.DebugContext(ctx, "ExtractThumbnail: ffmpeg output", "output", string(out))
	}
	if err != nil {
		return nil, fmt.Errorf("ExtractThumbnail: ffmpeg failed: %w\n%s", err, out)
	}

	return f.processExtractedFrame(outputPath)
}

func (f *Video2FrameExtractorImpl) processExtractedFrame(path string) (temp.Data, error) {
	frameFile, err := os.Open(path)
	defer helper.QuietClose(frameFile, f.slogger)

	if err != nil {
		return nil, fmt.Errorf("ExtractFrames: open frame %s: %w", path, err)
	}
	data, err := temp.DataTemp(frameFile)

	if err != nil {
		return nil, fmt.Errorf("ExtractFrames: read frame %s: %w", path, err)
	}
	return data, nil
}

func NewVideo2FrameExtractor(cfg *conf.FfmpegConfig) ocr.Video2FrameExtractor {
	return &Video2FrameExtractorImpl{
		slogger: slog.With("service", "Video2FrameExtractor"),
		cfg:     cfg,
	}
}
