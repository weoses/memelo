package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/youtube-service/conf"
)

// DownloadCallback receives the raw video stream and its mime type and must
// fully consume reader before returning. It returns whatever object the
// caller built from the stream (e.g. an uploaded S3BackedData handle).
type DownloadCallback[T any] func(ctx context.Context, reader io.Reader, mimeType string) (T, error)

type YouTubeDownloader[T any] interface {
	// Download fetches the video via the external download API and streams
	// the response body into callback. No temp file is created on disk.
	Download(ctx context.Context, url string, callback DownloadCallback[T]) (T, error)
}

type youTubeDownloaderImpl[T any] struct {
	cfg    *conf.YouTubeConfig
	client *http.Client
	log    *slog.Logger
}

func (d *youTubeDownloaderImpl[T]) Download(ctx context.Context, videoURL string, callback DownloadCallback[T]) (T, error) {
	var zero T
	jobID, err := d.createJob(ctx, videoURL)
	if err != nil {
		return zero, err
	}

	downloadURL, err := d.pollProgress(ctx, jobID)
	if err != nil {
		return zero, err
	}

	return d.downloadFile(ctx, downloadURL, callback)
}

func (d *youTubeDownloaderImpl[T]) createJob(ctx context.Context, videoURL string) (string, error) {
	params := url.Values{}
	params.Set("format", d.cfg.VideoFormat)
	params.Set("url", videoURL)
	params.Set("apikey", d.cfg.ApiKey)
	params.Set("max_duration", strconv.Itoa(d.cfg.MaxDuration))

	reqURL := "https://" + d.cfg.ApiHost + "/ajax/download.php?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("YouTubeDownloader: failed to build job request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("YouTubeDownloader: job request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		ID      string `json:"id"`
		Error   string `json:"error,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("YouTubeDownloader: failed to decode job response: %w", err)
	}
	if !result.Success {
		return "", fmt.Errorf("YouTubeDownloader: API rejected job: %s", result.Error)
	}

	d.log.InfoContext(ctx, "download job created", "job_id", result.ID)
	return result.ID, nil
}

func (d *youTubeDownloaderImpl[T]) pollProgress(ctx context.Context, jobID string) (string, error) {
	params := url.Values{}
	params.Set("id", jobID)
	pollURL := "https://" + d.cfg.ApiHost + "/ajax/progress?" + params.Encode()
	iteration := 0
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, pollURL, nil)
		if err != nil {
			return "", fmt.Errorf("YouTubeDownloader: failed to build progress request: %w", err)
		}

		resp, err := d.client.Do(req)
		if err != nil {
			return "", fmt.Errorf("YouTubeDownloader: progress request failed: %w", err)
		}

		var result struct {
			Success     int    `json:"success"`
			Progress    int    `json:"progress"`
			DownloadURL string `json:"download_url,omitempty"`
			Text        string `json:"text,omitempty"`
		}
		decodeErr := json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if decodeErr != nil {
			return "", fmt.Errorf("YouTubeDownloader: failed to decode progress response: %w", decodeErr)
		}

		if result.Success == 0 && result.Progress == 0 {
			return "", fmt.Errorf("YouTubeDownloader: job failed: %s", result.Text)
		}

		if iteration >= 100 {
			return "", fmt.Errorf("YouTubeDownloader: progress request timed out")
		}

		d.log.InfoContext(ctx, "download progress", "job_id", jobID, "progress", result.Progress)

		if result.Progress == 1000 {
			return result.DownloadURL, nil
		}
		iteration++
		time.Sleep(2 * time.Second)
	}
}

func (d *youTubeDownloaderImpl[T]) downloadFile(ctx context.Context, downloadURL string, callback DownloadCallback[T]) (T, error) {
	var zero T
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return zero, fmt.Errorf("YouTubeDownloader: failed to build file request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return zero, fmt.Errorf("YouTubeDownloader: file download failed: %w", err)
	}
	defer resp.Body.Close()

	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "video/mp4"
	}

	result, err := callback(ctx, resp.Body, mimeType)
	if err != nil {
		return zero, fmt.Errorf("YouTubeDownloader: callback failed: %w", err)
	}

	d.log.InfoContext(ctx, "video downloaded", "mime_type", mimeType)
	return result, nil
}

func NewYouTubeDownloader(cfg *conf.Config) (YouTubeDownloader[temp.S3BackedData], error) {
	return &youTubeDownloaderImpl[temp.S3BackedData]{
		cfg:    cfg.YouTube,
		client: &http.Client{},
		log:    slog.With("service", "YouTubeDownloader"),
	}, nil
}
