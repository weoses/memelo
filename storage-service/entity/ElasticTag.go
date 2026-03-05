package entity

import "github.com/google/uuid"

type ElasticTag struct {
	Id          uuid.UUID
	Tag         string
	Description string
	EmbeddingV1 *ElasticEmbeddingV1
	Created     int64
	Updated     int64
}
