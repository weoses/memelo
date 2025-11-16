package api

import (
	"context"

	"github.com/weoses/memelo/apispec/ocr-server/server"
	"github.com/weoses/memelo/ocr-server/service"
)

type Handler struct {
	imageSerivce service.ImageService
}

// PostApiV1OcrProcess implements server.StrictServerInterface.
func (h *Handler) PostApiV1OcrProcess(ctx context.Context, request server.PostApiV1OcrProcessRequestObject) (server.PostApiV1OcrProcessResponseObject, error) {
	response, err := h.imageSerivce.ProcessImage(ctx, request)
	if err != nil {
		return nil, err
	}

	return response, nil

}

func NewApiHandler(imageService service.ImageService) (server.StrictServerInterface, error) {
	return &Handler{
		imageSerivce: imageService,
	}, nil
}
