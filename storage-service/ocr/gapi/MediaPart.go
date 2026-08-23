package gapi

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genai"
)

// buildPart prefers a presigned S3 URL over uploading through this process --
// confirmed working (both generateContent and embedContent) against the
// Gemini Developer API (API-key auth) via a live test against a real object
// in the test bucket: Gemini fetches the presigned URL itself server-side,
// so this avoids reading the whole object through this process and
// re-uploading it. Falls back to the Files API when data isn't S3-backed or
// presigning fails (e.g. local/non-S3 storage).
func buildPart(ctx context.Context, client *genai.Client, data temp.Data, mimeType string, logger *slog.Logger) (*genai.Part, error) {
	if s3data, ok := data.(temp.S3BackedData); ok {
		if url, err := s3data.GetPresignedUrl(ctx); err == nil {
			logger.InfoContext(ctx, "buildPart: using presigned url", "url", url)
			return genai.NewPartFromURI(url, mimeType), nil
		}
	}

	reader, err := data.Reader(ctx)
	if err != nil {
		return nil, fmt.Errorf("read data: %w", err)
	}
	defer helper.QuietClose(reader, logger)

	ctx, span := tracer.Start(ctx, "gemini.upload_file", trace.WithAttributes(attribute.String("gemini.mime_type", mimeType)))
	defer span.End()

	file, err := client.Files.Upload(ctx, reader, &genai.UploadFileConfig{MIMEType: mimeType})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("upload file: %w", err)
	}

	return genai.NewPartFromURI(file.URI, mimeType), nil
}
