package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"syscall"

	"github.com/kkdai/youtube/v2"
	"github.com/weoses/memelo/youtube-service/conf"
)

type YouTubeDownloader interface {
	// Download fetches the best progressive format fitting MaxVideoSizeBytes to a temp file.
	// The caller is responsible for removing the returned file after use.
	Download(ctx context.Context, url string) (filePath string, mimeType string, err error)
}

type youTubeDownloaderImpl struct {
	cfg *conf.YouTubeConfig
	log *slog.Logger
}

func (d *youTubeDownloaderImpl) Download(ctx context.Context, url string) (string, string, error) {
	client := youtube.Client{}

	video, err := client.GetVideoContext(ctx, url)
	if err != nil {
		return "", "", fmt.Errorf("YouTubeDownloader: failed to fetch video info: %w", err)
	}

	format, err := d.selectFormat(video.Formats)
	if err != nil {
		return "", "", err
	}

	if err := d.checkDiskSpace(format.ContentLength); err != nil {
		return "", "", err
	}

	tmpFile, err := os.CreateTemp(d.cfg.TempDir, "memelo-yt-*.mp4")
	if err != nil {
		return "", "", fmt.Errorf("YouTubeDownloader: failed to create temp file: %w", err)
	}

	filePath := tmpFile.Name()

	stream, _, err := client.GetStreamContext(ctx, video, format)
	if err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(filePath)
		return "", "", fmt.Errorf("YouTubeDownloader: failed to get stream: %w", err)
	}
	defer stream.Close()

	if _, err := io.Copy(tmpFile, stream); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(filePath)
		return "", "", fmt.Errorf("YouTubeDownloader: failed to download video: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(filePath)
		return "", "", fmt.Errorf("YouTubeDownloader: failed to close temp file: %w", err)
	}

	d.log.InfoContext(ctx, "video downloaded", "path", filePath, "size", format.ContentLength, "quality", format.QualityLabel)
	return filePath, format.MimeType, nil
}

func (d *youTubeDownloaderImpl) selectFormat(formats youtube.FormatList) (*youtube.Format, error) {
	var candidates []youtube.Format
	for _, f := range formats {
		if f.AudioChannels > 0 && f.ContentLength > 0 && f.ContentLength <= d.cfg.MaxVideoSizeBytes {
			candidates = append(candidates, f)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf(
			"YouTubeDownloader: no progressive format fits within %d bytes (max size)",
			d.cfg.MaxVideoSizeBytes,
		)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Bitrate > candidates[j].Bitrate
	})

	return &candidates[0], nil
}

func (d *youTubeDownloaderImpl) checkDiskSpace(required int64) error {
	dir := d.cfg.TempDir
	if dir == "" {
		dir = os.TempDir()
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return fmt.Errorf("YouTubeDownloader: failed to check disk space: %w", err)
	}

	available := int64(stat.Bavail) * int64(stat.Bsize)
	if available < required {
		return fmt.Errorf(
			"YouTubeDownloader: insufficient disk space — need %d bytes, have %d bytes available",
			required, available,
		)
	}
	return nil
}

func NewYouTubeDownloader(cfg *conf.Config) (YouTubeDownloader, error) {
	return &youTubeDownloaderImpl{
		cfg: cfg.YouTube,
		log: slog.With("service", "YouTubeDownloader"),
	}, nil
}
