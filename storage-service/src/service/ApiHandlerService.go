package service

import (
	"context"
	"errors"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"mine.local/ocr-gallery/apispec/meme-storage/server"
	"mine.local/ocr-gallery/storage-service/entity"
	"mine.local/ocr-gallery/storage-service/helper"
)

type ApiHandler struct {
	metaStorage  MetadataStorageService
	imageStorage ImageStorageService
	ocr          OcrSerivce
	validate     *validator.Validate
}

// GetMemeImageThumbUrl implements server.StrictServerInterface.
func (a *ApiHandler) GetMemeImageThumbUrl(ctx context.Context, request server.GetMemeImageThumbUrlRequestObject) (server.GetMemeImageThumbUrlResponseObject, error) {
	memeMetadata, err := a.metaStorage.GetById(ctx, request.MemeId)
	if err != nil {
		return nil, err
	}

	if memeMetadata.AccountId != request.AccountId {
		return nil, echo.ErrNotFound
	}

	url, err := a.imageStorage.GetUrlThumb(ctx, memeMetadata.S3Id)
	if err != nil {
		return nil, err
	}

	resp := server.GetMemeImageThumbUrl200JSONResponse{}
	resp.Url = &url
	return resp, nil
}

// GetMemeImageUrl implements server.StrictServerInterface.
func (a *ApiHandler) GetMemeImageUrl(ctx context.Context, request server.GetMemeImageUrlRequestObject) (server.GetMemeImageUrlResponseObject, error) {
	memeMetadata, err := a.metaStorage.GetById(ctx, request.MemeId)
	if err != nil {
		return nil, err
	}

	if memeMetadata.AccountId != request.AccountId {
		return nil, echo.ErrNotFound
	}

	url, err := a.imageStorage.GetUrl(ctx, memeMetadata.S3Id)
	if err != nil {
		return nil, err
	}

	resp := server.GetMemeImageUrl200JSONResponse{}
	resp.Url = &url
	return resp, nil
}

// CreateMeme implements server.StrictServerInterface.
func (a *ApiHandler) CreateMeme(ctx context.Context, request server.CreateMemeRequestObject) (server.CreateMemeResponseObject, error) {
	image := request.Body

	idUuid, _ := uuid.NewRandom()
	if len(*image.ImageBase64) == 0 {
		return nil, errors.New("image is empty")
	}

	hash := helper.CalcHash(image.ImageBase64)

	hashDuplicate, err := a.metaStorage.GetByHash(ctx, hash)
	if err != nil {
		//TODO handle fail
		return nil, err
	}

	if hashDuplicate != nil {
		hashAccountDuplicate, err := a.metaStorage.GetByHashAndAccountId(ctx, request.AccountId, hash)
		if err != nil {
			//TODO handle fail
			return nil, err
		}

		response := server.CreateMeme200JSONResponse{}

		if hashAccountDuplicate == nil {
			copyMetadata := *hashDuplicate
			log.Printf("Found meme duplicate in another account: id=%s hash=%s", hashDuplicate.ImageId, hash)
			copyMetadata.ImageId = idUuid
			copyMetadata.AccountId = request.AccountId
			err = a.metaStorage.Save(ctx, &copyMetadata)
			if err != nil {
				return nil, err
			}
			helper.ElasticToCreateResponse(&copyMetadata, &response)
		} else {
			log.Printf("Found meme duplicate in this account: id=%s hash=%s", hashAccountDuplicate.ImageId, hash)
			helper.ElasticToCreateResponse(hashAccountDuplicate, &response)
		}

		return response, nil
	}

	reqImage := helper.ImageToEntity2(request.Body)
	ocrResult, err := a.ocr.DoOcr(ctx, idUuid, reqImage)
	if err != nil {
		return nil, err
	}

	err = a.imageStorage.Save(ctx, idUuid, ocrResult.Image, ocrResult.Thumbnail.Image)
	if err != nil {
		return nil, err
	}

	elasticMetaData := entity.ElasticImageMetaData{
		ImageId:   idUuid,
		S3Id:      idUuid,
		AccountId: request.AccountId,
		ThumbSize: &entity.ElasticSizes{
			Height: ocrResult.Thumbnail.Height,
			Width:  ocrResult.Thumbnail.Width,
		},
		Hash:   hash,
		Result: ocrResult.OcrText,
	}
	err = a.validate.Struct(elasticMetaData)
	if err != nil {
		//TODO handle fail
		return nil, err
	}

	err = a.metaStorage.Save(ctx, &elasticMetaData)
	if err != nil {
		//TODO handle fail
		return nil, err
	}

	response := server.CreateMeme200JSONResponse{}
	helper.ElasticToCreateResponse(&elasticMetaData, &response)
	return response, nil
}

// SearchMeme implements server.StrictServerInterface.
func (a *ApiHandler) SearchMeme(ctx context.Context, request server.SearchMemeRequestObject) (server.SearchMemeResponseObject, error) {
	query := request.Params.MemeQuery

	log.Printf("SearchMeme: query=%s", query)

	matchedMetadata, err := a.metaStorage.Search(
		ctx,
		request.AccountId,
		query,
		request.Params.SearchAfterId,
		request.Params.PageSize,
	)

	if err != nil {
		return nil, err
	}

	response := make(server.SearchMeme200JSONResponse, len(matchedMetadata))
	for index, metadataItem := range matchedMetadata {

		imageThumbUrl, err := a.imageStorage.GetUrlThumb(ctx, metadataItem.Metadata.S3Id)
		if err != nil {
			return nil, err
		}
		imageUrl, err := a.imageStorage.GetUrl(ctx, metadataItem.Metadata.S3Id)
		if err != nil {
			return nil, err
		}

		dto := server.SearchMemeDto{}
		helper.ElasticToSearchMemeDto(metadataItem, &dto)
		dto.ImageUrl = &imageUrl
		dto.Thumbnail = new(server.SearchMemeThumb)
		dto.Thumbnail.ThumbUrl = &imageThumbUrl
		dto.Thumbnail.ThumbHeight = &metadataItem.Metadata.ThumbSize.Height
		dto.Thumbnail.ThumbWidth = &metadataItem.Metadata.ThumbSize.Width
		response[index] = dto
	}

	return response, nil
}

func NewApiHandler(
	metaStorage MetadataStorageService,
	imageStorage ImageStorageService,
	ocr OcrSerivce,
	validator *validator.Validate,
) server.StrictServerInterface {
	return &ApiHandler{
		metaStorage:  metaStorage,
		imageStorage: imageStorage,
		ocr:          ocr,
		validate:     validator,
	}
}
