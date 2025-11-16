package service

import (
	"context"

	"github.com/h2non/bimg"
	"github.com/pkg/errors"
	"github.com/weoses/memelo/ocr-server/conf"
	"github.com/weoses/memelo/ocr-server/entity"
)

type ImageConveter interface {
	MakeEntity(ctx context.Context, data *[]byte) (*entity.Image, error)
	ConvertImageJPEG(ctx context.Context, image *entity.Image) (*entity.Image, error)
	MakeThumb(ctx context.Context, image *entity.Image) (*entity.Image, error)
}

type ImageConveterImpl struct {
	config *conf.ImageConverterConfig
}

// ConvertImageJPEG implements ImageConveter.
func (i *ImageConveterImpl) ConvertImageJPEG(ctx context.Context, image *entity.Image) (*entity.Image, error) {
	img := bimg.NewImage(*image.Data)
	size, err := img.Size()
	if err != nil {
		return nil, errors.Wrap(err, "Image Size() failed")
	}

	bytesData, err := img.Convert(bimg.JPEG)
	if err != nil {
		return nil, errors.Wrap(err, "Image Convert() to JPEG failed")
	}

	retImage := new(entity.Image)
	retImage.Data = &bytesData
	retImage.Width = size.Width
	retImage.Height = size.Height

	return retImage, nil
}

// MakeThumb implements ImageConveter.
func (i *ImageConveterImpl) MakeThumb(ctx context.Context, image *entity.Image) (*entity.Image, error) {
	img := bimg.NewImage(*image.Data)
	newWidth := i.config.ThumbSize
	newHeight := int(float64(i.config.ThumbSize) / float64(image.Width) * float64(image.Height))

	bytesData, err := img.Resize(newWidth, newHeight)
	if err != nil {
		return nil, errors.Wrap(err, "Image Resize() failed")
	}

	return &entity.Image{
		Data:   &bytesData,
		Width:  newWidth,
		Height: newHeight,
	}, nil
}

func (i *ImageConveterImpl) MakeEntity(ctx context.Context, data *[]byte) (*entity.Image, error) {
	img := bimg.NewImage(*data)
	size, err := img.Size()
	if err != nil {
		return nil, errors.Wrap(err, "Image Size() failed")
	}

	return &entity.Image{
		Data:   data,
		Width:  size.Width,
		Height: size.Height,
	}, nil
}

func NewImageConverter(config *conf.ImageConverterConfig) (ImageConveter, error) {
	return &ImageConveterImpl{
		config: config,
	}, nil
}
