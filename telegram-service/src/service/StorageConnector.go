package service

import (
	"bytes"
	"context"
	"encoding/base64"
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

	CreateMeme(file []byte, mime string, accountId uuid.UUID) (uuid.UUID, error)
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

// CreateMeme implements UploadService.
func (u *StorageConnectorImpl) CreateMeme(file []byte, mime string, accountId uuid.UUID) (uuid.UUID, error) {
	strbuf := bytes.NewBufferString("")
	encoder := base64.NewEncoder(base64.RawStdEncoding, strbuf)
	encoder.Write(file)
	encoder.Close()
	data := strbuf.String()

	reqBody := client.CreateMemeJSONRequestBody{}
	reqBody.ImageBase64 = &data
	reqBody.MimeType = &mime

	resp, err := u.cl.CreateMemeWithResponse(
		context.TODO(),
		accountId,
		reqBody,
	)

	if err != nil {
		return uuid.Nil, err
	}

	if resp.StatusCode() != 200 {
		return uuid.Nil, errors.New("UploadFile() - failed - storage service status code non 2xx ")
	}

	return *resp.JSON200.Id, nil
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
