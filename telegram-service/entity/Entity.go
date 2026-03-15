package entity

import "github.com/google/uuid"

type Choice struct {
}

type MemeCreateResult struct {
	Id              uuid.UUID
	Text            string
	DuplicateStatus string
}

type MemeSearchResult struct {
	Id          uuid.UUID
	SortId      string
	ImageUrl    string
	ThumbUrl    string
	ThumbWidth  int
	ThumbHeight int
}
