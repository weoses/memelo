package helper

import (
	"crypto/md5"
	"encoding/hex"

	"mine.local/ocr-gallery/apispec/meme-storage/server"
	"mine.local/ocr-gallery/apispec/ocr-server/client"
	"mine.local/ocr-gallery/storage-service/entity"
)

func CalcHash(base64Image *string) string {
	hasher := md5.New()
	hasher.Write([]byte(*base64Image))
	byteHash := hasher.Sum(nil)
	return hex.EncodeToString(byteHash)
}

func ElasticToCreateResponse(elasticEntity *entity.ElasticImageMetaData, dto *server.CreateMeme200JSONResponse) {
	dto.Hash = &elasticEntity.Hash
	dto.Id = &elasticEntity.ImageId
	dto.OcrResult = &elasticEntity.Result
}

func ElasticToSearchMemeDto(elasticEntity *entity.ElasticMatchedContent, dto *server.SearchMemeDto) {
	dto.OcrResult = &elasticEntity.Metadata.Result
	dto.Hash = &elasticEntity.Metadata.Hash
	dto.Id = &elasticEntity.Metadata.ImageId
	dto.SortId = &elasticEntity.Metadata.Created

	highlightText := *elasticEntity.ResultMatched
	dto.OcrResultHighlight = &highlightText
}

func ImageToEntity(image *client.ImageDto) *entity.Image {
	return &entity.Image{
		ImageBase64: image.ImageBase64,
		MimeType:    *image.MimeType,
	}
}

func ImageToEntity2(image *server.ImageDto) *entity.Image {
	return &entity.Image{
		ImageBase64: image.ImageBase64,
		MimeType:    *image.MimeType,
	}
}
