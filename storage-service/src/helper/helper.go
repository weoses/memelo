package helper

import (
	"crypto/md5"
	"encoding/hex"

	"mine.local/ocr-gallery/apispec/meme-storage/server"
	"mine.local/ocr-gallery/apispec/ocr-server/client"
	"mine.local/ocr-gallery/storage-service/entity"
)

func CalcHash(base64Image string) string {
	hasher := md5.New()
	hasher.Write([]byte(base64Image))
	byteHash := hasher.Sum(nil)
	return hex.EncodeToString(byteHash)
}

func ElasticToCreateResponse(
	elasticEntity *entity.ElasticImageMetaData,
	duplicate server.DuplicateStatus,
	dto *server.CreateMeme200JSONResponse,
) {
	dto.Hash = &elasticEntity.Hash
	dto.Id = &elasticEntity.ImageId
	dto.OcrResult = &elasticEntity.Result
	dto.DuplicateStatus = &duplicate
}

func OcrImageToEntity(image *client.ImageWithSizeDto) *entity.Image {
	return &entity.Image{
		ImageBase64: image.Image.ImageBase64,
		Width:       image.Width,
		Height:      image.Height,
	}
}

func ApiImageToEntity(image *server.CreateImageRequestDto) *entity.Image {
	return &entity.Image{
		ImageBase64: image.ImageBase64,
	}
}
