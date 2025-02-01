package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"strings"

	"mine.local/ocr-gallery/apispec/ocr-server/server"
	"mine.local/ocr-gallery/ocr-server/entity"
)

type ImageService interface {
	ProcessImage(ctx context.Context, image server.PostApiV1OcrProcessRequestObject) (server.PostApiV1OcrProcessResponseObject, error)
}

type ImageServiceImpl struct {
	ocr  OcrProcessor
	conv ImageConveter
}

// ProcessImage implements ImageService.
func (i *ImageServiceImpl) ProcessImage(ctx context.Context, image server.PostApiV1OcrProcessRequestObject) (server.PostApiV1OcrProcessResponseObject, error) {
	imageEnt := imageDtoToEntity(image.Body.Image)
	imageEnt, err := i.conv.ConvertImage(ctx, imageEnt)
	if err != nil {
		return nil, err
	}

	imageThumbEnt, imgThumbSizes, err := i.conv.MakeThumb(ctx, imageEnt)
	if err != nil {
		return nil, err
	}

	processorName := i.ocr.GetName()
	stringData, err := i.ocr.DoOcr(ctx, imageEnt)
	if err != nil {
		return nil, err
	}

	response := new(server.PostApiV1OcrProcess200JSONResponse)

	response.Image = imageEntityToDto(imageEnt)
	response.ImageThumb = new(server.ThumbnailDto)
	response.ImageThumb.Image = imageEntityToDto(imageThumbEnt)
	response.ImageThumb.Height = &imgThumbSizes.Height
	response.ImageThumb.Width = &imgThumbSizes.Width
	response.ImageText = &[]server.OcrResponseItem{
		{
			ProcessorKey: &processorName,
			Text:         &stringData,
		},
	}
	return response, nil

}

func imageDtoToEntity(dto *server.ImageDto) *entity.Image {
	decoder := base64.NewDecoder(base64.RawStdEncoding, strings.NewReader(*dto.ImageBase64))
	data, _ := io.ReadAll(decoder)

	retval := new(entity.Image)
	retval.Data = &data
	retval.MimeType = *dto.MimeType
	return retval
}

func imageEntityToDto(entity *entity.Image) *server.ImageDto {
	buff := bytes.NewBufferString("")

	encoder := base64.NewEncoder(base64.RawStdEncoding, buff)
	defer encoder.Close()
	encoder.Write(*entity.Data)
	encoder.Close()
	data := buff.String()

	retval := new(server.ImageDto)
	retval.ImageBase64 = &data
	retval.MimeType = &entity.MimeType
	return retval
}

func NewImageService(ocr OcrProcessor, conv ImageConveter) (ImageService, error) {
	return &ImageServiceImpl{
		ocr:  ocr,
		conv: conv,
	}, nil
}
