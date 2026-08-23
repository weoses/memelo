package ocr

import (
	"context"
	"fmt"

	"github.com/h2non/bimg"
	"github.com/pkg/errors"

	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/conf"
)

type ImageConveter interface {
	Convert2Jpeg(ctx context.Context, rawImage temp.Data) (temp.Data, error)
	MakeThumbnail(ctx context.Context, rawImage temp.Data) (temp.Data, error)
	GetSize(ctx context.Context, rawImage temp.Data) (int, int, error)
}

type ImageConveterImpl struct {
	config *conf.ImageConverterConfig
}

func (i *ImageConveterImpl) GetSize(ctx context.Context, rawImage temp.Data) (int, int, error) {
	imgData, err := rawImage.ReadAll(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("error reading image bytes: %w", err)
	}

	img := bimg.NewImage(imgData)
	size, err := img.Size()
	if err != nil {
		return 0, 0, fmt.Errorf("error getting size of image: %w", err)
	}
	return size.Width, size.Height, nil
}

func (i *ImageConveterImpl) Convert2Jpeg(ctx context.Context, rawImage temp.Data) (temp.Data, error) {
	imgData, err := rawImage.ReadAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("error reading image bytes: %w", err)
	}

	img := bimg.NewImage(imgData)

	bytesData, err := img.Convert(bimg.JPEG)
	if err != nil {
		return nil, errors.Wrap(err, "Image Convert() to JPEG failed")
	}
	return temp.DataBytes(bytesData), nil
}

func (i *ImageConveterImpl) MakeThumbnail(ctx context.Context, rawImage temp.Data) (temp.Data, error) {
	imgData, err := rawImage.ReadAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("error reading image bytes: %w", err)
	}
	img := bimg.NewImage(imgData)
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

	return temp.DataBytes(bytesData), nil
}

func NewImageConverter(cfg *conf.Config) (ImageConveter, error) {
	return &ImageConveterImpl{
		config: cfg.ImageConverter,
	}, nil
}
