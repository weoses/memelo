package entity

import "github.com/google/uuid"

type Choice struct {
}

type MemeCreateResult struct {
	Id              uuid.UUID
	Text            string
	Caption         string
	DuplicateStatus string
	Tags            []string
}

const ResultTypeImage = "image"
const ResultTypeVideo = "video"

type MemeSearchResult struct {
	Id string

	MediaUrl    string
	MediaWidth  int
	MediaHeight int

	ThumbUrl    string
	ThumbWidth  int
	ThumbHeight int

	Type string

	Caption string
}
