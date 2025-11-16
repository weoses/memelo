package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"mine.local/ocr-gallery/apispec/meme-storage/client"
	"mine.local/ocr-gallery/common/commonhelper"
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

	CreateMeme(ctx context.Context, file []byte, mime string, accountId uuid.UUID) (*entity.MemeCreateResult, error)

	DeleteMeme(ctx context.Context, accountId uuid.UUID, memeId uuid.UUID) error
}

type StorageConnectorImpl struct {
	cl client.ClientWithResponsesInterface
}

func (s *StorageConnectorImpl) DeleteMeme(ctx context.Context, accountId uuid.UUID, memeId uuid.UUID) error {
	response, err := s.cl.DeleteMemeWithResponse(ctx, accountId, memeId)
	if err != nil {
		return fmt.Errorf("DeleteMeme failed: accountId=%s memeId=%s %w", accountId, memeId, err)
	}

	if response.StatusCode() >= 400 {
		return fmt.Errorf("DeleteMeme failed: accountId=%s memeId=%s storage_status=%d", accountId, memeId, response.StatusCode())
	}
	return nil
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
			Query:             query,
			PageSize:          &pageSize,
			SearchAfterSortId: searchAfterSortId,
		})

	if err != nil {
		return nil, fmt.Errorf("storageService: search meme query failed query: %s : %w", query, err)
	}

	if response.HTTPResponse.StatusCode == 500 {
		errorMessage := response.JSON500.Error
		return nil, fmt.Errorf("storageService: failed to request storage service: %v", errorMessage)
	}

	if response.HTTPResponse.StatusCode >= 400 {
		return nil, fmt.Errorf("storageService: failed to request storage service")
	}

	entityResult := make([]*entity.MemeSearchResult, len(*response.JSON200))
	for i, dto := range *response.JSON200 {
		entityResult[i] = &entity.MemeSearchResult{
			Id:          *dto.Id,
			ImageUrl:    dto.Image.Url,
			ThumbUrl:    dto.Thumbnail.Url,
			ThumbWidth:  dto.Thumbnail.Width,
			ThumbHeight: dto.Thumbnail.Height,
			SortId:      *dto.SortId,
		}
	}
	return entityResult, err
}

// CreateMeme implements UploadService.
func (u *StorageConnectorImpl) CreateMeme(ctx context.Context, file []byte, mime string, accountId uuid.UUID) (*entity.MemeCreateResult, error) {
	strbuf := bytes.NewBufferString("")
	encoder := base64.NewEncoder(base64.RawStdEncoding, strbuf)
	encoder.Write(file)
	encoder.Close()
	data := strbuf.String()

	reqBody := client.CreateMemeJSONRequestBody{}
	reqBody.ImageBase64 = data

	response, err := u.cl.CreateMemeWithResponse(
		ctx,
		accountId,
		reqBody,
	)

	if err != nil {
		return nil, fmt.Errorf("storageService: create meme failed: %w", err)
	}

	if response.HTTPResponse.StatusCode == 500 {
		errorMessage := response.JSON500.Error
		return nil, fmt.Errorf("storageService: failed to request storage service: %s", commonhelper.DefaultString(errorMessage))
	}

	if response.HTTPResponse.StatusCode >= 400 {
		return nil, fmt.Errorf("storageService: failed to request storage service")
	}
	creationResult := &entity.MemeCreateResult{
		Id:   response.JSON200.Id,
		Text: response.JSON200.OcrResult,
	}
	creationResult.DuplicateStatus = string(response.JSON200.DuplicateStatus)

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
