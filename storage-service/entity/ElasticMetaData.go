package entity

import "github.com/google/uuid"

type MetadataType string

const ImageMetadataType MetadataType = "image"
const VideoMetadataType MetadataType = "video"

type ElasticImageMetaData struct {
	ImageId   uuid.UUID `validator:"required"`
	S3Id      uuid.UUID `validator:"required"`
	AccountId uuid.UUID `validator:"required"`
	Hash      string
	Result    string
	ThumbSize *Sizes `validator:"required"`
	ImageSize *Sizes
	Created   int64 `validator:"required"`
	Updated   int64

	EmbeddingList []EmbeddingItem

	Tags []string

	Type MetadataType `validator:"required"`

	ResultData *Result

	ResultPerVideoSlices []ResultPerVideoSlice

	ManuallyUpdated bool
}
