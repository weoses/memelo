package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/adrg/strutil/metrics"
	"github.com/weoses/memelo/common/commonhelper"

	"github.com/adrg/strutil"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/weoses/memelo/apispec/meme-storage/server"
	"github.com/weoses/memelo/common/commonconst"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/helper"
)

type ApiHandler struct {
	metaStorage  MetadataStorageService
	imageStorage ImageStorageService
	ocr          OcrSerivce
	validate     *validator.Validate
}

func (a *ApiHandler) DeleteMeme(ctx context.Context, request server.DeleteMemeRequestObject) (server.DeleteMemeResponseObject, error) {
	item, err := a.metaStorage.GetById(ctx, request.AccountId, request.MemeId)
	if err != nil {
		return nil, err
	}

	if item == nil {
		return server.DeleteMeme200Response{}, nil
	}

	err = a.metaStorage.DeleteById(ctx, item.AccountId, item.ImageId)
	if err != nil {
		return nil, err
	}

	err = a.imageStorage.DeleteImage(ctx, item.S3Id)
	if err != nil {
		return nil, err
	}
	return server.DeleteMeme200Response{}, nil
}

// CreateMeme implements server.StrictServerInterface.
func (a *ApiHandler) CreateMeme(
	ctx context.Context,
	request server.CreateMemeRequestObject,
) (server.CreateMemeResponseObject, error) {
	image := request.Body
	accountId := request.AccountId

	slog.Info("CreateMeme",
		commonconst.ACCOUNTID_LOG, accountId)

	idUuid, _ := uuid.NewRandom()
	if len(image.ImageBase64) == 0 {
		return nil, errors.New("image is empty")
	}

	hash := helper.CalcHash(request.Body.ImageBase64)
	hashDuplicate, err := a.findHashDuplicates(ctx, accountId, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to find hash duplicates : %w", err)
	}

	if hashDuplicate != nil && len(hashDuplicate) > 0 {
		return a.HandleDuplicate(ctx, server.DuplicateHash, hashDuplicate[0], request)
	}

	reqImage := helper.ApiImageToEntity(request.Body)
	ocrResult, err := a.ocr.DoOcr(ctx, reqImage)
	if err != nil {
		return nil, fmt.Errorf("failed to do ocr : %w", err)
	}

	ocrTextResult := ocrResult.OcrText
	slog.Info("CreateMeme: ocr result",
		commonconst.ACCOUNTID_LOG, accountId,
		"id", idUuid,
		"ocrText", ocrTextResult)

	contentDuplicate, err := a.findContentDuplicates(ctx, accountId, ocrResult)
	if err != nil {
		return nil, err
	}

	//if strings.TrimSpace(ocrTextResult) == "" {
	//	return nil, errors.New("no text on image")
	//}

	if contentDuplicate != nil {
		contentDuplicateTextResult := contentDuplicate.Result
		similarity := strutil.Similarity(ocrTextResult, contentDuplicateTextResult, metrics.NewLevenshtein())

		slog.Info("CreateMeme: found content-duplicate by embedding search",
			commonconst.ACCOUNTID_LOG, accountId,
			"id", idUuid,
			"dupId", contentDuplicate.ImageId,
			"ocrText", ocrTextResult,
			"dupOcrText", contentDuplicate.Result,
			"similarity", similarity)

		if similarity > 0.5 {
			return a.HandleDuplicate(ctx, server.DuplicateImage, contentDuplicate, request)
		}
	}

	err = a.imageStorage.Save(ctx, idUuid, ocrResult.Image.ImageBase64, ocrResult.Thumbnail.ImageBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to save image metadata : %w", err)
	}

	elasticMetaData := OcrResultToElastic(
		idUuid,
		accountId,
		hash,
		time.Now().UnixMicro(),
		ocrResult,
	)

	err = a.validate.Struct(elasticMetaData)
	if err != nil {
		//TODO handle fail
		return nil, err
	}

	err = a.metaStorage.Save(ctx, elasticMetaData)
	if err != nil {
		//TODO handle fail
		return nil, err
	}

	response := server.CreateMeme200JSONResponse{}
	helper.ElasticToCreateResponse(elasticMetaData, server.New, &response)
	return response, nil
}

// SearchMeme implements server.StrictServerInterface.
func (a *ApiHandler) SearchMeme(ctx context.Context, request server.SearchMemeRequestObject) (server.SearchMemeResponseObject, error) {
	query := request.Params.Query
	accountId := request.AccountId
	searchAfter := request.Params.SearchAfterSortId
	pageSize := request.Params.PageSize

	slog.Info("SearchMeme", "query", query)
	matchedMetadata, err := a.searchMeme(ctx, accountId, query, searchAfter, pageSize)
	if err != nil {
		return nil, fmt.Errorf("searchMeme failed: %w", err)
	}

	slog.Info("SearchMeme results",
		commonconst.ACCOUNTID_LOG, request.AccountId,
		"query", query,
		"resultListSize", len(matchedMetadata))

	response := make(server.SearchMeme200JSONResponse, len(matchedMetadata))
	for index, metadataItem := range matchedMetadata {
		slog.Debug("Search meme result item",
			commonconst.ACCOUNTID_LOG, metadataItem.AccountId,
			"id", metadataItem.ImageId,
			"s3id", metadataItem.S3Id,
			"index", index)

		imageThumbUrl, err := a.imageStorage.GetUrlThumb(ctx, metadataItem.S3Id)
		if err != nil {
			return nil, fmt.Errorf("failed to obtain thumb url for image id=%s : %w", metadataItem.ImageId, err)
		}
		imageUrl, err := a.imageStorage.GetUrl(ctx, metadataItem.S3Id)
		if err != nil {
			return nil, fmt.Errorf("failed to obtain image url for image id=%s : %w", metadataItem.ImageId, err)
		}

		dto := server.SearchMemeResponseItemDto{
			Hash: &metadataItem.Hash,
			Id:   &metadataItem.ImageId,
			Image: &server.ImageUrlDto{
				Url:    imageUrl,
				Height: metadataItem.ImageSize.Height,
				Width:  metadataItem.ImageSize.Width,
			},
			Thumbnail: &server.ImageUrlDto{
				Url:    imageThumbUrl,
				Height: metadataItem.ThumbSize.Height,
				Width:  metadataItem.ThumbSize.Width,
			},
			OcrResult: &metadataItem.Result,
			SortId:    &metadataItem.Created,
		}
		response[index] = dto
	}

	return response, nil
}

func (a *ApiHandler) searchMeme(
	ctx context.Context,
	accountId uuid.UUID,
	query string,
	searchAfter *int64,
	pageSize *int,
) ([]*entity.ElasticImageMetaData, error) {
	if query == "" {
		slog.Info("SearchMeme method=ALL")
		matchedMetadataAll, err := a.metaStorage.SearchAll(
			ctx,
			accountId,
			searchAfter,
			pageSize,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to search memes[ALL] : %w", err)
		}
		return matchedMetadataAll, nil
	}

	asId, err := uuid.Parse(query)
	if err == nil {
		slog.Info("SearchMeme method=ID")
		matchedById, err := a.metaStorage.GetById(ctx, accountId, asId)
		if err != nil {
			return nil, fmt.Errorf("failed to search memes[ID] : %w", err)
		}

		if matchedById != nil {
			return []*entity.ElasticImageMetaData{
				matchedById,
			}, nil
		}
	}

	slog.Info("SearchMeme method=SIMPLE")
	matchedMetadataSimple, err := a.metaStorage.SearchSimple(
		ctx,
		accountId,
		query,
		searchAfter,
		pageSize,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search memes[SIMPLE] : %w", err)
	}

	if len(matchedMetadataSimple) > 0 || searchAfter != nil {
		return matchedMetadataSimple, nil
	}

	slog.Info("SearchMeme method=FUZZY")
	matchedMetadataFuzzy, err := a.metaStorage.SearchFuzzy(
		ctx,
		accountId,
		query,
		pageSize,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search memes[Fuzzy] : %w", err)
	}

	return matchedMetadataFuzzy, nil
}

// CheckDuplicates implements server.StrictServerInterface.
func (a *ApiHandler) CheckDuplicates(ctx context.Context, request server.CheckDuplicatesRequestObject) (server.CheckDuplicatesResponseObject, error) {
	return server.CheckDuplicates200Response{},
		a.iterateDocuments(
			ctx,
			request.AccountId,
			func(ctx2 context.Context, emc *entity.ElasticImageMetaData) error {
				err := a.internalCheckDuplicate(ctx2, emc)
				if err != nil {
					slog.Error("internalCheckDuplicate failed",
						"ImageId", emc.ImageId,
						commonconst.ERR_LOG, err)
				}

				return nil
			})

}

// UpdateOcr implements server.StrictServerInterface.
func (a *ApiHandler) UpdateOcr(ctx context.Context, request server.UpdateOcrRequestObject) (server.UpdateOcrResponseObject, error) {
	return server.UpdateOcr200Response{},
		a.iterateDocuments(
			ctx,
			request.AccountId,
			func(ctx context.Context, emc *entity.ElasticImageMetaData) error {
				err := a.internalUpdateOcr(ctx, emc)
				if err != nil {
					slog.Error("internalUpdateOcr failed",
						"ImageId", emc.ImageId,
						commonconst.ERR_LOG, err)
				}
				return nil
			})
}

// GetMemeImageThumbUrl implements server.StrictServerInterface.
func (a *ApiHandler) GetMemeImageThumbUrl(ctx context.Context, request server.GetMemeImageThumbUrlRequestObject) (server.GetMemeImageThumbUrlResponseObject, error) {
	memeMetadata, err := a.metaStorage.GetById(ctx, request.AccountId, request.MemeId)
	if err != nil {
		return nil, fmt.Errorf("storage GetById failed: %w", err)
	}

	if memeMetadata == nil {
		return nil, fmt.Errorf("image for accountId=%s and MemeId=%s not found", request.AccountId, request.MemeId)
	}

	if memeMetadata.AccountId != request.AccountId {
		return nil, fmt.Errorf("image for accountId=%s and MemeId=%s not found", request.AccountId, request.MemeId)
	}

	url, err := a.imageStorage.GetUrlThumb(ctx, memeMetadata.S3Id)
	if err != nil {
		return nil, err
	}

	resp := &server.GetMemeImageThumbUrl200JSONResponse{
		Url:    url,
		Height: memeMetadata.ThumbSize.Height,
		Width:  memeMetadata.ThumbSize.Width,
	}
	return resp, nil
}

// GetMemeImageUrl implements server.StrictServerInterface.
func (a *ApiHandler) GetMemeImageUrl(ctx context.Context, request server.GetMemeImageUrlRequestObject) (server.GetMemeImageUrlResponseObject, error) {
	memeMetadata, err := a.metaStorage.GetById(ctx, request.AccountId, request.MemeId)
	if err != nil {
		return nil, fmt.Errorf("storage GetById failed: %w", err)
	}

	if memeMetadata == nil {
		return nil, fmt.Errorf("image for accountId=%s and MemeId=%s not found", request.AccountId, request.MemeId)
	}

	if memeMetadata.AccountId != request.AccountId {
		return nil, fmt.Errorf("image for accountId=%s and MemeId=%s not found", request.AccountId, request.MemeId)
	}

	url, err := a.imageStorage.GetUrl(ctx, memeMetadata.S3Id)
	if err != nil {
		return nil, err
	}

	resp := &server.GetMemeImageUrl200JSONResponse{
		Url:    url,
		Height: memeMetadata.ImageSize.Height,
		Width:  memeMetadata.ImageSize.Width,
	}
	return resp, nil
}

// UpdateOcrOne implements server.StrictServerInterface.
func (a *ApiHandler) UpdateOcrOne(ctx context.Context, request server.UpdateOcrOneRequestObject) (server.UpdateOcrOneResponseObject, error) {
	memeMetadata, err := a.metaStorage.GetById(ctx, request.AccountId, request.MemeId)
	if err != nil {
		return nil, fmt.Errorf("storage GetById failed: %w", err)
	}

	if memeMetadata == nil {
		return nil, fmt.Errorf("image for accountId=%s and MemeId=%s not found", request.AccountId, request.MemeId)
	}

	if memeMetadata.AccountId != request.AccountId {
		return nil, fmt.Errorf("image for accountId=%s and MemeId=%s not found", request.AccountId, request.MemeId)
	}

	err = a.internalUpdateOcr(ctx, memeMetadata)
	if err != nil {
		return nil, fmt.Errorf("internalUpdateOcr failed: %w", err)
	}

	return server.UpdateOcrOne200Response{}, nil
}

func (a *ApiHandler) internalCheckDuplicate(ctx context.Context, emc *entity.ElasticImageMetaData) error {
	id := emc.ImageId
	embedding := emc.EmbeddingV1
	slog.Info("Check-duplicate",
		"id", id.String())

	if embedding == nil {
		return fmt.Errorf("embedding is nil. id=%s", id)
	}

	embeddingFoundImage, err := a.metaStorage.GetByEmbeddingV1(ctx, emc.AccountId, embedding, 10)
	if err != nil {
		return fmt.Errorf("GetByEmbeddingV1All failed: %w", err)
	}

	for i, item := range embeddingFoundImage {
		if i == 0 {
			continue
		}

		err := a.metaStorage.DeleteById(ctx, item.AccountId, item.ImageId)
		if err != nil {
			slog.Warn("Check-duplicate: failed to delete duplicate image",
				"error", err,
				"id", item.ImageId)
		}
	}

	return nil
}

func (a *ApiHandler) internalUpdateOcr(ctx context.Context, emc *entity.ElasticImageMetaData) error {
	id := emc.ImageId
	accountId := emc.AccountId
	hash := emc.Hash
	s3id := emc.S3Id
	created := emc.Created

	slog.Info("UpdateOcr: checking image",
		"id", id)

	base64ImgData, err := a.imageStorage.GetImage(ctx, s3id)
	if err != nil {
		return fmt.Errorf("storage GetImage failed: %w", err)
	}

	item := &entity.Image{
		ImageBase64: *base64ImgData,
		Width:       emc.ImageSize.Width,
		Height:      emc.ImageSize.Height,
	}

	ocrResult, err := a.ocr.DoOcr(ctx, item)
	if err != nil {
		return fmt.Errorf("doOcr failed: %w", err)
	}

	elasticObject := OcrResultToElastic(id, accountId, hash, created, ocrResult)
	err = a.metaStorage.Save(ctx, elasticObject)
	if err != nil {
		return fmt.Errorf("doOcr failed: %w", err)
	}

	return nil
}

func (a *ApiHandler) iterateDocuments(
	ctx context.Context,
	accountId uuid.UUID,
	callback func(context.Context, *entity.ElasticImageMetaData) error,
) error {
	items, err := a.metaStorage.SearchAll(ctx, accountId, nil, commonhelper.Addr(1000))
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	for len(items) > 0 {
		for _, item := range items {
			err = callback(ctx, item)
			if err != nil {
				return fmt.Errorf("iterateDocuments callback failed: %w", err)
			}
		}

		if len(items) > 0 {
			items, err = a.metaStorage.SearchAll(ctx, accountId, &items[len(items)-1].Created, commonhelper.Addr(1000))
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}
		}
	}

	return nil
}

func (a *ApiHandler) findHashDuplicates(
	ctx context.Context,
	accountId uuid.UUID,
	hash string,
) ([]*entity.ElasticImageMetaData, error) {
	return a.metaStorage.GetByHash(ctx, accountId, hash, nil)
}

func (a *ApiHandler) findContentDuplicates(
	ctx context.Context,
	accountId uuid.UUID,
	ocrResult *OcrProcessedResult,
) (*entity.ElasticImageMetaData, error) {
	embedding := &entity.ElasticEmbeddingV1{
		Data:  &ocrResult.Embedding.Data,
		Model: ocrResult.Embedding.Model,
	}
	results, err := a.metaStorage.GetByEmbeddingV1(ctx, accountId, embedding, 1)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}
	return results[0], err
}

func (a *ApiHandler) HandleDuplicate(
	ctx context.Context,
	status server.DuplicateStatus,
	duplicate *entity.ElasticImageMetaData,
	request server.CreateMemeRequestObject,
) (server.CreateMemeResponseObject, error) {
	response := server.CreateMeme200JSONResponse{}
	if duplicate.AccountId != request.AccountId {
		copyMetadata := *duplicate
		slog.Info("Found meme duplicate in another account", "id", duplicate.ImageId)
		copyMetadata.ImageId, _ = uuid.NewRandom()
		copyMetadata.AccountId = request.AccountId
		err := a.metaStorage.Save(ctx, &copyMetadata)
		if err != nil {
			return nil, err
		}
		helper.ElasticToCreateResponse(&copyMetadata, status, &response)
	} else {
		slog.Info("Found meme duplicate in this account", "id", duplicate.ImageId)
		helper.ElasticToCreateResponse(duplicate, status, &response)
	}

	return response, nil
}

func OcrResultToElastic(idUuid uuid.UUID, accountId uuid.UUID, hash string, created int64, ocrResult *OcrProcessedResult) *entity.ElasticImageMetaData {
	return &entity.ElasticImageMetaData{
		ImageId:   idUuid,
		S3Id:      idUuid,
		AccountId: accountId,
		ImageSize: &entity.ElasticSizes{
			Height: ocrResult.Image.Height,
			Width:  ocrResult.Image.Width,
		},
		ThumbSize: &entity.ElasticSizes{
			Height: ocrResult.Thumbnail.Height,
			Width:  ocrResult.Thumbnail.Width,
		},
		Created: created,
		Hash:    hash,
		Result:  ocrResult.OcrText,
		EmbeddingV1: &entity.ElasticEmbeddingV1{
			Data:  &ocrResult.Embedding.Data,
			Model: ocrResult.Embedding.Model,
		},
	}
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
