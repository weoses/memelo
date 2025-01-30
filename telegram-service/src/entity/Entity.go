package entity

import "github.com/google/uuid"

type Choice struct {
}

type MongoTgUserAccountBinding struct {
	UserId    int64
	AccountId uuid.UUID
}
