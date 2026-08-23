package ffmpeg

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/ffmpeg-service/conf"
)

func testCfg() *conf.FfmpegConfig {
	return &conf.FfmpegConfig{FfmpegBinary: "ffmpeg", FfprobeBinary: "ffprobe"}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// minimalVideo generates a tiny H.264 raw MPEG-TS video of the given duration
// (in whole seconds) so tests don't depend on any fixture files.
func minimalVideo(t *testing.T, durationSec int) temp.Data {
	t.Helper()

	f, err := os.CreateTemp("", "test-video-*.mp4")
	if err != nil {
		t.Fatalf("minimalVideo: create temp file: %v", err)
	}
	f.Close()
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	cmd := exec.Command("ffmpeg",
		"-f", "lavfi", "-i", fmt.Sprintf("color=c=0x336699:s=32x32:r=2:d=%d", durationSec),
		// force every frame to be a keyframe so downstream "-c copy" slicing can seek/cut freely
		"-c:v", "libx264", "-g", "1", "-keyint_min", "1",
		"-t", fmt.Sprintf("%d", durationSec), "-y", "-loglevel", "quiet",
		f.Name(),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("minimalVideo: ffmpeg: %v\n%s", err, out)
	}

	data, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatalf("minimalVideo: read: %v", err)
	}
	return temp.DataBytes(data)
}

func TestConvertToMp4(t *testing.T) {
	video := minimalVideo(t, 1)
	defer video.Close(context.Background())

	out, err := ConvertToMp4(context.Background(), testCfg(), video, testLogger())
	if err != nil {
		t.Fatalf("ConvertToMp4: %v", err)
	}
	defer out.Close(context.Background())

	size, err := out.Size(context.Background())
	if err != nil {
		t.Fatalf("Size: %v", err)
	}
	if size == 0 {
		t.Fatal("ConvertToMp4 produced an empty file")
	}
}

func TestExtractOneFrame(t *testing.T) {
	video := minimalVideo(t, 1)
	defer video.Close(context.Background())

	frame, err := ExtractOneFrame(context.Background(), testCfg(), video, testLogger())
	if err != nil {
		t.Fatalf("ExtractOneFrame: %v", err)
	}
	defer frame.Close(context.Background())

	data, err := frame.ReadAll(context.Background())
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("ExtractOneFrame produced an empty file")
	}
	// JPEG magic bytes.
	if data[0] != 0xFF || data[1] != 0xD8 {
		t.Fatalf("ExtractOneFrame did not produce a JPEG, got magic bytes %x %x", data[0], data[1])
	}
}

func TestSliceVideoWithOverlap(t *testing.T) {
	video := minimalVideo(t, 6)
	defer video.Close(context.Background())

	slices, err := SliceVideoWithOverlap(context.Background(), testCfg(), video, 3*time.Second, 1*time.Second, testLogger())
	if err != nil {
		t.Fatalf("SliceVideoWithOverlap: %v", err)
	}
	defer closeAll(context.Background(), slices)

	if len(slices) == 0 {
		t.Fatal("SliceVideoWithOverlap produced no slices")
	}
	for i, s := range slices {
		if s.EndTime <= s.StartTime {
			t.Errorf("slice %d: end (%d) <= start (%d)", i, s.EndTime, s.StartTime)
		}
		size, err := s.Data.Size(context.Background())
		if err != nil || size == 0 {
			t.Errorf("slice %d: empty or unreadable (size=%d, err=%v)", i, size, err)
		}
	}
}

func TestSliceVideoWithOverlap_InvalidIntervalOverlap(t *testing.T) {
	video := minimalVideo(t, 2)
	defer video.Close(context.Background())

	_, err := SliceVideoWithOverlap(context.Background(), testCfg(), video, 1*time.Second, 1*time.Second, testLogger())
	if err == nil {
		t.Fatal("expected error when interval <= overlap")
	}
}
