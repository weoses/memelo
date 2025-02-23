package entity

import "github.com/google/uuid"

type Choice struct {
}

type MongoTgUserAccountBinding struct {
	UserId    int64
	AccountId uuid.UUID
}

type MemeCreateResult struct {
	Id              uuid.UUID
	Text            string
	DuplicateStatus string
}

type MemeSearchResult struct {
	Id          uuid.UUID
	SortId      int64
	ImageUrl    string
	ThumbUrl    string
	ThumbWidth  int
	ThumbHeight int
}
