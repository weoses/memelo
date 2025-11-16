package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-playground/validator/v10"
	"mine.local/ocr-gallery/apispec/ocr-server/client"
	"mine.local/ocr-gallery/storage-service/conf"
	"mine.local/ocr-gallery/storage-service/entity"
	"mine.local/ocr-gallery/storage-service/helper"
)

type OcrSerivce interface {
	DoOcr(ctx context.Context, incomingImage *entity.Image) (*OcrProcessedResult, error)
}

type OcrServiceImpl struct {
	ocrclient client.ClientWithResponsesInterface
	validate  *validator.Validate
}

type OcrProcessedResult struct {
	OcrText   string
	Thumbnail *entity.Image `validator:required`
	Image     *entity.Image `validator:required`
	Embedding *OcrEmbedding `validator:required`
}

type OcrEmbedding struct {
	Data  []float32 `validator:required`
	Model string
}

func (ocr *OcrServiceImpl) DoOcr(
	ctx context.Context,
	incomingImage *entity.Image,
) (*OcrProcessedResult, error) {
	request := client.OcrRequestDto{
		Image: &client.ImageDto{
			ImageBase64: incomingImage.ImageBase64,
		},
	}

	response, err := ocr.ocrclient.PostApiV1OcrProcessWithResponse(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("ocr request failed: %w", err)
	}

	if response.StatusCode() == 500 {
		return nil, fmt.Errorf("ocr request failed status=%s message=%v", response.Status(), response.JSON500.Error)
	}

	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("ocr request failed status=%s", response.Status())
	}

	responseJson := response.JSON200

	textVariants := responseJson.ImageText
	sourceImage := responseJson.ImageSource
	thumbImage := responseJson.ImageThumb

	retval := &OcrProcessedResult{
		OcrText:   textVariantsToString(textVariants),
		Image:     helper.OcrImageToEntity(sourceImage),
		Thumbnail: helper.OcrImageToEntity(thumbImage),
		Embedding: &OcrEmbedding{
			Data:  *responseJson.Embedding.Data,
			Model: *responseJson.Embedding.ModelName,
		},
	}
	return retval, ocr.validate.Struct(retval)
}

func textVariantsToString(textVariants *[]client.OcrResponseItem) string {
	builder := strings.Builder{}
	for _, textVariant := range *textVariants {
		builder.WriteString(*textVariant.Text)
		builder.WriteString(" ")
	}
	return builder.String()
}

func NewOcrService(conf *conf.OcrConfig, validate *validator.Validate) (OcrSerivce, error) {
	ocrServiceUrl := conf.Uri
	slog.Info("Creating ocr service",
		"url", ocrServiceUrl)

	client, err := client.NewClientWithResponses(ocrServiceUrl)
	if err != nil {
		return nil, err
	}
	return &OcrServiceImpl{
			ocrclient: client,
			validate:  validate,
		},
		nil
}
