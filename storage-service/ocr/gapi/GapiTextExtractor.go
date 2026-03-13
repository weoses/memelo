package gapi

import (
	"bytes"
	"context"

	vision "cloud.google.com/go/vision/apiv1"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/ocr"
	"google.golang.org/api/option"
)

type GcloudTextExtractorImpl struct {
	client *vision.ImageAnnotatorClient
}

// GetName implements TextExtractor.
func (m *GcloudTextExtractorImpl) GetName() string {
	return "GCloud"
}

func (m *GcloudTextExtractorImpl) DoOcr(ctx context.Context, image []byte) (string, error) {
	img, err := vision.NewImageFromReader(bytes.NewReader(image))
	if err != nil {
		return "", err
	}

	texts, err := m.client.DetectTexts(ctx, img, nil, 100)
	if err != nil {
		return "", err
	}

	if len(texts) > 0 {
		return texts[0].Description, nil
	}
	return "", nil
}

func NewOcrProcessor(ocrConf *conf.ImageOcrConfig) (ocr.TextExtractor, error) {
	visionClient, err := vision.NewImageAnnotatorClient(
		context.Background(),
		option.WithEndpoint(ocrConf.ApiEndpoint),
	)
	if err != nil {
		return nil, err
	}

	return &GcloudTextExtractorImpl{
		client: visionClient,
	}, nil
}
