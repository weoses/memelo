package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"

	"github.com/google/uuid"
	"mine.local/ocr-gallery/apispec/meme-storage/client"
	"mine.local/ocr-gallery/telegram-service/conf"
	"mine.local/ocr-gallery/telegram-service/entity"
)

type StorageConnector interface {
	ProcessSearchQuery(
		ctx context.Context,
		accountId uuid.UUID,
		query string,
		pageSize int,
		searchAfterSortId *int64,
	) ([]*entity.MemeSearchResult, error)

	CreateMeme(file []byte, mime string, accountId uuid.UUID) (*entity.MemeCreateResult, error)
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
	searchAfterSortId *int64,
) ([]*entity.MemeSearchResult, error) {

	response, err := s.cl.SearchMemeWithResponse(
		ctx,
		accountId,
		&client.SearchMemeParams{
			MemeQuery:         query,
			PageSize:          &pageSize,
			SearchAfterSortId: searchAfterSortId,
		})

	if err != nil {
		return nil, err
	}

	if response.HTTPResponse.StatusCode >= 400 {
		return nil, errors.New("failed to request storage service")
	}

	entityResult := make([]*entity.MemeSearchResult, len(*response.JSON200))
	for i, dto := range *response.JSON200 {
		entityResult[i] = &entity.MemeSearchResult{
			Id:          *dto.Id,
			ImageUrl:    *dto.ImageUrl,
			ThumbUrl:    *dto.Thumbnail.ThumbUrl,
			ThumbWidth:  *dto.Thumbnail.ThumbWidth,
			ThumbHeight: *dto.Thumbnail.ThumbHeight,
			SortId:      *dto.SortId,
		}
	}
	return entityResult, err
}

// CreateMeme implements UploadService.
func (u *StorageConnectorImpl) CreateMeme(file []byte, mime string, accountId uuid.UUID) (*entity.MemeCreateResult, error) {
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
		return nil, err
	}

	if resp.StatusCode() != 200 {
		return nil, errors.New("UploadFile() - failed - storage service status code non 2xx ")
	}

	creationResult := &entity.MemeCreateResult{
		Id:   *resp.JSON200.Id,
		Text: *resp.JSON200.OcrResult,
	}

	if resp.JSON200.DuplicateStatus != nil {
		creationResult.DuplicateStatus = string(*resp.JSON200.DuplicateStatus)
	}

	return creationResult, nil
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
