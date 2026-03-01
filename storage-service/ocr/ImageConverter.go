package ocr

import (
	"context"
	"fmt"

	"github.com/h2non/bimg"
	"github.com/pkg/errors"
	"github.com/weoses/memelo/storage-service/conf"
)

type ImageConveter interface {
	ProcessOriginalImage(ctx context.Context, rawImage []byte) ([]byte, error)
	MakeThumbnail(ctx context.Context, rawImage []byte) ([]byte, error)
	GetSize(ctx context.Context, rawImage []byte) (int, int, error)
}

type ImageConveterImpl struct {
	config *conf.ImageConverterConfig
}

func (i *ImageConveterImpl) GetSize(ctx context.Context, rawImage []byte) (int, int, error) {
	img := bimg.NewImage(rawImage)
	size, err := img.Size()
	if err != nil {
		return 0, 0, fmt.Errorf("error getting size of image: %w", err)
	}
	return size.Width, size.Height, nil
}

func (i *ImageConveterImpl) ProcessOriginalImage(ctx context.Context, rawImage []byte) ([]byte, error) {
	img := bimg.NewImage(rawImage)

	bytesData, err := img.Convert(bimg.JPEG)
	if err != nil {
		return nil, errors.Wrap(err, "Image Convert() to JPEG failed")
	}
	return bytesData, nil
}

func (i *ImageConveterImpl) MakeThumbnail(ctx context.Context, rawImage []byte) ([]byte, error) {
	img := bimg.NewImage(rawImage)
	size, err := img.Size()
	if err != nil {
		return nil, fmt.Errorf("error getting size of image: %w", err)
	}

	newWidth := i.config.ThumbSize
	newHeight := int(float64(i.config.ThumbSize) / float64(size.Width) * float64(size.Height))

	bytesData, err := img.Resize(newWidth, newHeight)
	if err != nil {
		return nil, errors.Wrap(err, "Image Resize() failed")
	}

	return bytesData, nil
}

func NewImageConverter(config *conf.ImageConverterConfig) (ImageConveter, error) {
	return &ImageConveterImpl{
		config: config,
	}, nil
}
