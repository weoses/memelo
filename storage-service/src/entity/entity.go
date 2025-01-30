package entity

import "github.com/google/uuid"

type OcrResult struct {
	ProcessorKey string
	Text         string
}

type ElasticImageMetaData struct {
	ImageId   uuid.UUID
	S3Id      uuid.UUID
	AccountId uuid.UUID
	Hash      string
	Result    string
	Created   int64
}

type ElasticMatchedContent struct {
	Metadata      *ElasticImageMetaData
	ResultMatched *[]string
}

type OcrProcessedResult struct {
	OcrText   string
	Thumbnail *Image
	Image     *Image
}

type Image struct {
	ImageBase64 *string
	MimeType    string
}
