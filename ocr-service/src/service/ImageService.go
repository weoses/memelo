package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"mine.local/ocr-gallery/apispec/ocr-server/server"
	"mine.local/ocr-gallery/common/commonerror"
	"mine.local/ocr-gallery/ocr-server/entity"
)

type ImageService interface {
	ProcessImage(ctx context.Context, image server.PostApiV1OcrProcessRequestObject) (server.PostApiV1OcrProcessResponseObject, error)
}

type ImageServiceImpl struct {
	ocr     OcrProcessor
	conv    ImageConveter
	comarer ImageEmbeddingExtractor
}

func (i *ImageServiceImpl) validateProcessImage(ctx context.Context, image server.PostApiV1OcrProcessRequestObject) error {
	if image.Body == nil || image.Body.Image == nil || image.Body.Image.ImageBase64 == "" {
		return commonerror.ApiError{
			StatusCode: http.StatusBadRequest,
			Message:    "No request body or image",
		}
	}
	return nil
}

// ProcessImage implements ImageService.
func (i *ImageServiceImpl) ProcessImage(ctx context.Context, image server.PostApiV1OcrProcessRequestObject) (server.PostApiV1OcrProcessResponseObject, error) {
	err := i.validateProcessImage(ctx, image)
	if err != nil {
		return nil, err
	}

	imageEntIncoming, err := i.convertImageDto(ctx, image.Body.Image)
	if err != nil {
		return nil, errors.Wrap(err, "convertImageDto failed")
	}

	imageEnt, err := i.conv.ConvertImageJPEG(ctx, imageEntIncoming)
	if err != nil {
		return nil, errors.Wrap(err, "ConvertImageJPEG failed: ")
	}

	imageThumbEnt, err := i.conv.MakeThumb(ctx, imageEnt)
	if err != nil {
		return nil, errors.Wrap(err, "MakeThumb failed")
	}

	processorName := i.ocr.GetName()
	stringData, err := i.ocr.DoOcr(ctx, imageEnt)
	if err != nil {
		return nil, errors.Wrap(err, "DoOcr failed")
	}

	embedding, err := i.comarer.GetImageEmbeddingV1(ctx, imageEnt)
	if err != nil {
		return nil, errors.Wrap(err, "GetImageEmbeddingV1 failed")
	}

	resultImgSource, err := i.imageEntityToDto(imageEnt)
	if err != nil {
		return nil, errors.Wrap(err, "imageEntityToDto for source image failed")
	}

	resultImgThumb, err := i.imageEntityToDto(imageThumbEnt)
	if err != nil {
		return nil, errors.Wrap(err, "imageEntityToDto for thumb image failed")
	}

	response := &server.PostApiV1OcrProcess200JSONResponse{
		ImageSource: resultImgSource,
		ImageThumb:  resultImgThumb,
		ImageText: &[]server.OcrResponseItem{
			{
				ProcessorKey: &processorName,
				Text:         &stringData,
			},
		},
		Embedding: &server.EmbeddingDto{
			ModelName: &embedding.Model,
			Data:      &embedding.Data,
		},
	}
	return response, nil
}

func pixels(uintArr []uint16) []int {
	intArr := make([]int, len(uintArr))
	for i, v := range uintArr {
		intArr[i] = int(v)
	}
	return intArr
}

func (i *ImageServiceImpl) convertImageDto(ctx context.Context, dto *server.ImageDto) (*entity.Image, error) {
	decoder := base64.NewDecoder(base64.RawStdEncoding, strings.NewReader(dto.ImageBase64))
	data, err := io.ReadAll(decoder)
	if err != nil {
		return nil, errors.Wrap(err, "image base64 decoding failed")
	}

	img, err := i.conv.MakeEntity(ctx, &data)
	if err != nil {
		return nil, errors.Wrap(err, "MakeEntity failed")
	}

	return img, nil
}

func (i *ImageServiceImpl) imageEntityToDto(entity *entity.Image) (*server.ImageWithSizeDto, error) {
	buff := bytes.NewBufferString("")

	encoder := base64.NewEncoder(base64.RawStdEncoding, buff)
	defer encoder.Close()
	_, err := encoder.Write(*entity.Data)
	if err != nil {
		return nil, errors.Wrap(err, "image base64 encoding failed")
	}
	encoder.Close()
	data := buff.String()

	return &server.ImageWithSizeDto{
		Image: server.ImageDto{
			ImageBase64: data,
		},
		Width:  entity.Width,
		Height: entity.Height,
	}, nil
}

func NewImageService(ocr OcrProcessor, conv ImageConveter, comp ImageEmbeddingExtractor) (ImageService, error) {
	return &ImageServiceImpl{
		ocr:     ocr,
		conv:    conv,
		comarer: comp,
	}, nil
}
