package entity

import "github.com/google/uuid"

type ElasticTag struct {
	Id          uuid.UUID
	AccountId   uuid.UUID
	Tag         string
	Description string

	Embedding *EmbeddingItem
	Created   int64
	Updated   int64
}
