package ocr

import (
	"bytes"
	"context"

	vision "cloud.google.com/go/vision/apiv1"
	"github.com/weoses/memelo/storage-service/conf"
	"google.golang.org/api/option"
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

func NewOcrProcessor(ocrConf *conf.ImageOcrConfig) (Img2TextService, error) {
	visionClient, err := vision.NewImageAnnotatorClient(
		context.Background(),
		option.WithEndpoint(ocrConf.ApiEndpoint),
	)
	if err != nil {
		return nil, err
	}

	return &Img2TextGcloudServiceImpl{
		client: visionClient,
	}, nil
}
