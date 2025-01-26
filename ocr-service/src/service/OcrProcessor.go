package service

import (
	"bytes"
	"context"

	vision "cloud.google.com/go/vision/apiv1"
	"mine.local/ocr-gallery/ocr-server/entity"
)

type OcrProcessor interface {
	GetName() string
	DoOcr(ctx context.Context, image *entity.Image) (string, error)
}

type OcrProcessorGcloudImpl struct {
	client *vision.ImageAnnotatorClient
}

// GetName implements OcrProcessor.
func (m *OcrProcessorGcloudImpl) GetName() string {
	return "GCloud"
}

func (m *OcrProcessorGcloudImpl) DoOcr(ctx context.Context, image *entity.Image) (string, error) {
	img, err := vision.NewImageFromReader(bytes.NewReader(*image.Data))
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

func NewVisionImageClient() (*vision.ImageAnnotatorClient, error) {
	return vision.NewImageAnnotatorClient(context.Background())
}

func NewOcrProcessor(visionClient *vision.ImageAnnotatorClient) (OcrProcessor, error) {
	return &OcrProcessorGcloudImpl{
		client: visionClient,
	}, nil
}
