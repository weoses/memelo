package entity

import "github.com/google/uuid"

type OcrResult struct {
	ProcessorKey string
	Text         string
}

type ElasticSizes struct {
	Width  int `validator:required`
	Height int `validator:required`
}
type ElasticImageMetaData struct {
	ImageId   uuid.UUID `validator:required`
	S3Id      uuid.UUID `validator:required`
	AccountId uuid.UUID `validator:required`
	Hash      string
	Result    string
	ThumbSize *ElasticSizes `validator:required`
	Created   int64         `validator:required`
}

type ElasticMatchedContent struct {
	Metadata      *ElasticImageMetaData `validator:required`
	ResultMatched *[]string             `validator:required`
}

type OcrThumbnail struct {
	Image  *Image `validator:required`
	Width  int    `validator:required`
	Height int    `validator:required`
}

type OcrProcessedResult struct {
	OcrText   string
	Thumbnail *OcrThumbnail `validator:required`
	Image     *Image        `validator:required`
}

type Image struct {
	ImageBase64 *string `validator:required`
	MimeType    string
}
