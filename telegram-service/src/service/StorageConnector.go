package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"mine.local/ocr-gallery/apispec/meme-storage/client"
	"mine.local/ocr-gallery/telegram-service/conf"
)

type StorageConnector interface {
	ProcessSearchQuery(
		ctx context.Context,
		accountId uuid.UUID,
		query string,
		pageSize int,
		searchAfter *uuid.UUID,
	) (*[]client.SearchMemeDto, error)
}

type StorageConnectorImpl struct {
	cl client.ClientWithResponsesInterface
}

// ProcessSearchQuery implements StorageConnector.
func (s *StorageConnectorImpl) ProcessSearchQuery(
	ctx context.Context,
	accountId uuid.UUID,
	query string,
	pageSize int,
	searchAfter *uuid.UUID,
) (*[]client.SearchMemeDto, error) {

	response, err := s.cl.SearchMemeWithResponse(
		ctx,
		accountId,
		&client.SearchMemeParams{
			MemeQuery:     query,
			PageSize:      &pageSize,
			SearchAfterId: searchAfter,
		})

	if err != nil {
		return nil, err
	}

	if response.HTTPResponse.StatusCode >= 400 {
		return nil, errors.New("failed to request storage service")
	}

	return response.JSON200, err
}

func NewStorageConnector(config *conf.StorageConfig) (StorageConnector, error) {
	cl, err := client.NewClientWithResponses(config.Uri)
	if err != nil {
		return nil, err
	}

	return &StorageConnectorImpl{
			cl: cl,
		},
		nil

}
