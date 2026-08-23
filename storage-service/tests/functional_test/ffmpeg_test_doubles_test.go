package functional_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/ocr"
)

// The functional test suite exercises the real video pipeline end to end, so
// it needs working ocr.Video2Mp4Converter / ocr.VideoSlicer /
// ocr.Video2FrameExtractor implementations. ffmpeg-service is a separate Go
// module (its own go.mod), so it can't be imported here — these are trimmed,
// test-only local-exec reimplementations wired in via fx.Decorate, assuming
// a real ffmpeg/ffprobe binary on the test machine (as CI already provides).

type localFfmpegVideo2Mp4 struct{}

var _ ocr.Video2Mp4Converter = (*localFfmpegVideo2Mp4)(nil)

func (localFfmpegVideo2Mp4) ConvertToMp4(ctx context.Context, video temp.Data) (temp.Data, error) {
	dir, err := os.MkdirTemp("", "test-video2mp4-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	inputPath := filepath.Join(dir, "input")
	if err := writeToFile(ctx, video, inputPath); err != nil {
		return nil, err
	}

	outputPath := filepath.Join(dir, "output.mp4")
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", inputPath, "-c:v", "libx264", "-c:a", "aac", "-movflags", "+faststart", outputPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg convert: %w\n%s", err, out)
	}
	return readFile(outputPath)
}

type localFfmpegFrameExtractor struct{}

var _ ocr.Video2FrameExtractor = (*localFfmpegFrameExtractor)(nil)

func (localFfmpegFrameExtractor) ExtractOneFrame(ctx context.Context, video temp.Data) (temp.Data, error) {
	dir, err := os.MkdirTemp("", "test-video2thumb-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	inputPath := filepath.Join(dir, "input.mp4")
	if err := writeToFile(ctx, video, inputPath); err != nil {
		return nil, err
	}

	outputPath := filepath.Join(dir, "thumb.jpg")
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", inputPath, "-vframes", "1", "-f", "image2", outputPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg extract frame: %w\n%s", err, out)
	}
	return readFile(outputPath)
}

type localFfmpegVideoSlicer struct{}

var _ ocr.VideoSlicer = (*localFfmpegVideoSlicer)(nil)

func (localFfmpegVideoSlicer) SliceVideoWithOverlap(ctx context.Context, video temp.Data, interval, overlap time.Duration) ([]ocr.VideoSlice, error) {
	if interval <= overlap {
		return nil, fmt.Errorf("interval (%s) must be greater than overlap (%s)", interval, overlap)
	}

	dir, err := os.MkdirTemp("", "test-videoslicer-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	inputPath := filepath.Join(dir, "input.mp4")
	if err := writeToFile(ctx, video, inputPath); err != nil {
		return nil, err
	}

	durOut, err := exec.CommandContext(ctx, "ffprobe", "-v", "error",
		"-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", inputPath).Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe: %w", err)
	}
	var durSec float64
	if _, err := fmt.Sscanf(string(durOut), "%f", &durSec); err != nil {
		return nil, fmt.Errorf("parse duration: %w", err)
	}
	total := time.Duration(durSec * float64(time.Second))

	step := interval - overlap
	var slices []ocr.VideoSlice
	for start := time.Duration(0); start < total; start += step {
		end := start + interval
		if end > total {
			end = total
		}
		if (end-start) < overlap && start > 0 {
			break
		}
		segPath := filepath.Join(dir, fmt.Sprintf("segment_%05d.mp4", len(slices)))
		cmd := exec.CommandContext(ctx, "ffmpeg",
			"-ss", fmt.Sprintf("%f", start.Seconds()),
			"-i", inputPath,
			"-t", fmt.Sprintf("%f", interval.Seconds()),
			"-c", "copy", segPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			for _, s := range slices {
				helper.QuietCloseCtx(ctx, s.Data, slog.Default())
			}
			return nil, fmt.Errorf("ffmpeg slice: %w\n%s", err, out)
		}
		data, err := readFile(segPath)
		if err != nil {
			return nil, err
		}
		slices = append(slices, ocr.VideoSlice{Data: data, StartTime: int(start.Seconds()), EndTime: int(end.Seconds())})
	}
	return slices, nil
}

func writeToFile(ctx context.Context, video temp.Data, path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	r, err := video.Reader(ctx)
	if err != nil {
		return err
	}
	defer r.Close()

	_, err = io.Copy(f, r)
	return err
}

func readFile(path string) (temp.Data, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return temp.DataTemp(f)
}
