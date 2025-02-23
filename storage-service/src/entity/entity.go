package entity

import "github.com/google/uuid"

type ElasticSizes struct {
	Width  int `validator:required`
	Height int `validator:required`
}

type ElasticImageMetaData struct {
	ImageId     uuid.UUID `validator:required`
	S3Id        uuid.UUID `validator:required`
	AccountId   uuid.UUID `validator:required`
	Hash        string
	Result      string
	ThumbSize   *ElasticSizes `validator:required`
	Created     int64         `validator:required`
	Updated     int64
	EmbeddingV1 *ElasticEmbeddingV1 `validator:required`
}

type ElasticEmbeddingV1 struct {
	Data  *[]float32 `validator:required`
	Model string     `validator:required`
}

type ElasticMatchedContent struct {
	Metadata      *ElasticImageMetaData `validator:required`
	ResultMatched *[]string             `validator:required`
}

type Image struct {
	ImageBase64 *string `validator:required`
	MimeType    string
}
