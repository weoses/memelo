package ocr

import (
	"bytes"
	"context"

	vision "cloud.google.com/go/vision/apiv1"
)

type Img2TextService interface {
	GetName() string
	DoOcr(ctx context.Context, image []byte) (string, error)
}

type Img2TextGcloudServiceImpl struct {
	client *vision.ImageAnnotatorClient
}

// GetName implements Img2TextService.
func (m *Img2TextGcloudServiceImpl) GetName() string {
	return "GCloud"
}

func (m *Img2TextGcloudServiceImpl) DoOcr(ctx context.Context, image []byte) (string, error) {
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

func NewVisionImageClient() (*vision.ImageAnnotatorClient, error) {
	return vision.NewImageAnnotatorClient(context.Background())
}

func NewOcrProcessor(visionClient *vision.ImageAnnotatorClient) (Img2TextService, error) {
	return &Img2TextGcloudServiceImpl{
		client: visionClient,
	}, nil
}
