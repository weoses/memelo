//go:generate go run -modfile=../tools/go.mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=ocr-server-config.yml ocr-server-api.yml
package api

import (
	"context"
	"encoding/base64"
	"log"
	"strings"

	mapper "github.com/dranikpg/dto-mapper"
	"mine.local/ocr-gallery/ocr-server/service"
)

type StrictServerInterfaceImpl struct {
	ocrService service.OcrService
}

func NewServerInterfaceImpl(ocrService service.OcrService) ServerInterface {
	return NewStrictHandler(
		&StrictServerInterfaceImpl{
			ocrService: ocrService,
		},
		[]StrictMiddlewareFunc{})
}

func (srv *StrictServerInterfaceImpl) PostApiV1OcrProcess(
	ctx context.Context,
	request PostApiV1OcrProcessRequestObject,
) (PostApiV1OcrProcessResponseObject, error) {
	base64DecodedReader := base64.NewDecoder(
		base64.StdEncoding,
		strings.NewReader(*request.Body.Image))

	log.Printf("Incoming Post-Ocr-Preprocess: Id=%s Name=%s", *request.Body.ImageId, *request.Body.ImageName)

	response, err := srv.ocrService.ProcessImage(*request.Body.ImageId, base64DecodedReader)
	if err != nil {
		return nil, err
	}

	log.Printf("Incoming Post-Ocr-Preprocess completed: Id=%s Name=%s", *request.Body.ImageId, *request.Body.ImageName)

	responseDto := PostApiV1OcrProcess200JSONResponse{}
	mapper.Map(&responseDto, response)

	return responseDto, nil
}
