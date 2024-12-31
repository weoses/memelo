package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"log"

	"github.com/google/uuid"
	"mine.local/ocr-gallery/image-collector/api/memestorage"
	"mine.local/ocr-gallery/image-collector/entity"
)

type MemeStorageApiService struct {
	metaStorage  MetadataStorageService
	imageStorage ImageStorageService
	ocr          OcrSerivce
}

// CreateMeme implements memestorage.StrictServerInterface.
func (a *MemeStorageApiService) CreateMeme(ctx context.Context, request memestorage.CreateMemeRequestObject) (memestorage.CreateMemeResponseObject, error) {
	image := request.Body.Data
	imageName := request.Body.Filename

	log.Printf("CreateMeme: Filename=%s", *imageName)

	hash := calcHash(image)

	log.Printf("Checking meme duplicates: hash=%s", hash)
	metadata, err := a.metaStorage.GetByHash(ctx, hash)
	if err != nil {
		//TODO handle fail
		return nil, err
	}

	if metadata != nil {
		log.Printf("Found meme duplicate: id=%s hash=%s", metadata.Id, hash)
		return &memestorage.CreateMeme200JSONResponse{
			Hash:      &hash,
			Id:        &metadata.Id,
			OcrResult: ocrResultToArray(&metadata.Result),
		}, nil
	}

	id, _ := uuid.NewRandom()
	idStr := id.String()

	log.Printf("Saving meme image: imageName=%s", *imageName)
	storageId, err := a.imageStorage.Save(ctx, image)
	if err != nil {
		//TODO handle fail
		return nil, err
	}

	fileMetaData := entity.StorageMetaData{
		Id:   storageId,
		Name: *imageName,
		Hash: hash,
	}

	log.Printf("Ocring meme: storageId=%s imageName=%s", storageId, *imageName)
	ocrResult, err := a.ocr.DoOcr(ctx, &fileMetaData, image)
	if err != nil {
		//TODO handle fail
		return nil, err
	}

	elasticMetadata := entity.ElasticImageMetaData{
		Storage: fileMetaData,
		Result:  *ocrResult,
		Id:      idStr,
	}

	log.Printf("Saving meme metadata: id=%s storageId=%s imageName=%s", idStr, storageId, *imageName)
	err = a.metaStorage.Save(ctx, &elasticMetadata)
	if err != nil {
		//TODO handle fail
		return nil, err
	}

	log.Printf("Meme processed: id=%s storageId=%s imageName=%s", idStr, storageId, *imageName)
	return &memestorage.CreateMeme200JSONResponse{
		Hash:      &hash,
		Id:        &idStr,
		OcrResult: ocrResultToArray(ocrResult),
	}, nil
}

// GetMemeImage implements memestorage.StrictServerInterface.
func (a *MemeStorageApiService) GetMemeImage(ctx context.Context, request memestorage.GetMemeImageRequestObject) (memestorage.GetMemeImageResponseObject, error) {
	log.Printf("GetMemeImage: id=%s", request.MemeId)
	metadata, err := a.metaStorage.GetById(ctx, request.MemeId)
	if err != nil {
		return nil, err
	}

	imgBase64, err := a.imageStorage.Get(ctx, metadata.Storage.Id)
	if err != nil {
		return nil, err
	}

	return memestorage.GetMemeImage200JSONResponse{
		Filename: &metadata.Storage.Name,
		Data:     imgBase64,
	}, err
}

// GetMemeInfo implements memestorage.StrictServerInterface.
func (a *MemeStorageApiService) GetMemeInfo(ctx context.Context, request memestorage.GetMemeInfoRequestObject) (memestorage.GetMemeInfoResponseObject, error) {
	panic("unimplemented")
}

func calcHash(image *string) string {
	hasher := md5.New()
	hasher.Write([]byte(*image))
	byteHash := hasher.Sum(nil)
	return hex.EncodeToString(byteHash)
}

func ocrResultToArray(ocrResult *entity.OcrResultBulk) *[]string {
	ocrResultStr := make([]string, len(*ocrResult.Texts))

	for index, item := range *ocrResult.Texts {
		ocrResultStr[index] = item.Text
	}
	return &ocrResultStr
}

func NewMemeStorageApiService(
	metaStorage MetadataStorageService,
	imageStorage ImageStorageService,
	ocr OcrSerivce,
) memestorage.StrictServerInterface {

	return &MemeStorageApiService{
		metaStorage:  metaStorage,
		imageStorage: imageStorage,
		ocr:          ocr,
	}
}
