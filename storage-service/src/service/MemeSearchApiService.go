package service

import (
	"context"
	"log"

	"mine.local/ocr-gallery/image-collector/api/memesearch"
)

type MemeSearchApiService struct {
	metaStorage MetadataStorageService
}

// SearchMeme implements memesearch.StrictServerInterface.
func (m *MemeSearchApiService) SearchMeme(ctx context.Context, request memesearch.SearchMemeRequestObject) (memesearch.SearchMemeResponseObject, error) {
	log.Printf("SearchMeme: query=%s", request.Params.MemeQuery)

	metadata, err := m.metaStorage.Search(ctx, request.Params.MemeQuery)
	if err != nil {
		return nil, err
	}

	response := make(memesearch.SearchMeme200JSONResponse, len(metadata))
	for index, meta := range metadata {
		response[index] = memesearch.MemeDto{
			Hash:      &meta.Storage.Hash,
			Id:        &meta.Id,
			OcrResult: ocrResultToArray(&meta.Result),
		}
	}

	return response, nil
}

func NewMemeSearchApiService(
	metaStorage MetadataStorageService,
) memesearch.StrictServerInterface {

	return &MemeSearchApiService{
		metaStorage: metaStorage,
	}
}
