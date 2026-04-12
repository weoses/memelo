package gapi

import (
	"context"
	"fmt"
	"log/slog"

	vision "cloud.google.com/go/vision/apiv1"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"
	pb "google.golang.org/genproto/googleapis/cloud/vision/v1"

	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/ocr"
	"golang.org/x/time/rate"
	"google.golang.org/api/option"
)

const ocrRatePerSecond = 25

type GcloudTextExtractorImpl struct {
	client  *vision.ImageAnnotatorClient
	limiter *rate.Limiter
	slogger *slog.Logger
}

// GetName implements Image2TextExtractor.
func (m *GcloudTextExtractorImpl) GetName() string {
	return "GCloud"
}

func (m *GcloudTextExtractorImpl) DoOcr(ctx context.Context, image temp.Data) (string, error) {
	m.slogger.InfoContext(ctx, "DoOcr start")

	if err := m.limiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limiter: %w", err)
	}

	var img *pb.Image
	if s3data, ok := image.(temp.S3BackedData); ok && s3data.IsGsSupported() {
		if s3path, pathErr := s3data.GetS3Url(ctx); pathErr == nil {
			if gcsUri, ok := toGcsUri(s3path); ok {
				m.slogger.InfoContext(ctx, "DoOcr using gcsUri", "uri", gcsUri)
				img = vision.NewImageFromURI(gcsUri)
			}
		}
	}
	if img == nil {
		m.slogger.InfoContext(ctx, "DoOcr using base64")
		reader, err := image.Reader()
		if err != nil {
			return "", err
		}
		defer helper.QuietClose(reader, m.slogger)
		img, err = vision.NewImageFromReader(reader)
		if err != nil {
			return "", err
		}
	}

	texts, err := m.client.DetectTexts(ctx, img, nil, 100)
	if err != nil {
		return "", err
	}

	if len(texts) > 0 {
		m.slogger.InfoContext(ctx, "DoOcr done", "chars", len(texts[0].Description))
		return texts[0].Description, nil
	}

	m.slogger.InfoContext(ctx, "DoOcr done", "chars", 0)
	return "", nil
}

func NewOcrProcessor(cfg *conf.Config) (ocr.Image2TextExtractor, error) {
	visionClient, err := vision.NewImageAnnotatorClient(
		context.Background(),
		option.WithEndpoint(cfg.ImageOcr.ApiEndpoint),
	)
	if err != nil {
		return nil, err
	}

	return &GcloudTextExtractorImpl{
		client:  visionClient,
		limiter: rate.NewLimiter(ocrRatePerSecond, ocrRatePerSecond/2),
		slogger: slog.With("service", "GcloudTextExtractor"),
	}, nil
}
