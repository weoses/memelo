package service

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"mine.local/ocr-gallery/apispec/ocr-server/client"
	"mine.local/ocr-gallery/storage-service/conf"
	"mine.local/ocr-gallery/storage-service/entity"
	"mine.local/ocr-gallery/storage-service/helper"
)

type OcrSerivce interface {
	DoOcr(ctx context.Context,
		id uuid.UUID,
		incomingImage *entity.Image,
	) (*OcrProcessedResult, error)
}

type OcrServiceImpl struct {
	ocrclient client.ClientWithResponsesInterface
	validate  *validator.Validate
}

type OcrProcessedResult struct {
	OcrText   string
	Thumbnail *OcrThumbnail `validator:required`
	Image     *entity.Image `validator:required`
	Embedding *OcrEmbedding
}
type OcrThumbnail struct {
	Image  *entity.Image `validator:required`
	Width  int           `validator:required`
	Height int           `validator:required`
}

type OcrEmbedding struct {
	Data  []float32 `validator:required`
	Model string
}

func (ocr *OcrServiceImpl) DoOcr(
	ctx context.Context,
	id uuid.UUID,
	incomingImage *entity.Image,
) (*OcrProcessedResult, error) {

	idStr := id.String()

	request := client.OcrRequestDto{
		ImageId: &idStr,
		Image: &client.ImageDto{
			ImageBase64: incomingImage.ImageBase64,
			MimeType:    &incomingImage.MimeType,
		},
	}

	response, err := ocr.ocrclient.PostApiV1OcrProcessWithResponse(ctx, request)
	if err != nil {
		return nil, err

	}

	if response.StatusCode() != 200 {
		return nil, errors.New("status code fault")
	}

	responseJson := response.JSON200

	textVariants := responseJson.ImageText

	image := responseJson.Image

	retval := new(OcrProcessedResult)
	retval.OcrText = textVariantsToString(textVariants)
	retval.Image = helper.ImageToEntity(image)

	if responseJson.ImageThumb != nil {
		thumbnail := responseJson.ImageThumb
		retval.Thumbnail = new(OcrThumbnail)
		retval.Thumbnail.Image = helper.ImageToEntity(thumbnail.Image)
		retval.Thumbnail.Width = *thumbnail.Width
		retval.Thumbnail.Height = *thumbnail.Height
	}

	if responseJson.Embedding != nil {
		retval.Embedding = new(OcrEmbedding)
		retval.Embedding.Data = *responseJson.Embedding.Data
		retval.Embedding.Model = *responseJson.Embedding.ModelName
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
	log.Printf("Creating ocr service url=%s\n", ocrServiceUrl)

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
