package openrouter

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"

	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"
)

// resolveMediaURL prefers a presigned S3 URL over inlining the media as a
// base64 data URL -- avoids reading the whole object through this process
// and re-uploading it to OpenRouter, mirroring the same S3-first pattern
// gapi's buildMediaPart uses for Gemini. Falls back to a data URL when data
// isn't S3-backed or presigning fails (e.g. local/non-S3 storage).
func resolveMediaURL(ctx context.Context, data temp.Data, mimeType string, logger *slog.Logger) (string, error) {
	if s3data, ok := data.(temp.S3BackedData); ok {
		if url, err := s3data.GetPresignedUrl(ctx); err == nil {
			logger.InfoContext(ctx, "resolveMediaURL: using presigned url", "url", url)
			return url, nil
		}
	}
	return toDataURL(ctx, data, mimeType, logger)
}

func toDataURL(ctx context.Context, data temp.Data, mimeType string, logger *slog.Logger) (string, error) {
	r, err := data.Reader(ctx)
	if err != nil {
		return "", fmt.Errorf("read data: %w", err)
	}
	defer helper.QuietClose(r, logger)
	raw, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read all: %w", err)
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}
