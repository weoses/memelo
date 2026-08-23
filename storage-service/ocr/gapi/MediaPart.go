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

// buildPart uploads data to Gemini's Files API and returns a Part referencing
// the uploaded file. This is the only mechanism confirmed to work reliably
// with the Gemini Developer API (API-key auth, which this package uses):
// Cloud Storage gs:// URIs require Vertex AI/OAuth, and Google does not
// document reliable support for arbitrary public/presigned URLs (e.g. our S3
// presigned URLs) on the generateContent endpoint this SDK calls. So every
// caller uploads rather than branching between upload/presigned-URL/inline
// depending on which method happened to be invoked.
func buildPart(ctx context.Context, client *genai.Client, data temp.Data, mimeType string, logger *slog.Logger) (*genai.Part, error) {
	reader, err := data.Reader()
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
