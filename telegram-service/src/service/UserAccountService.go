package service

import (
	"context"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"mine.local/ocr-gallery/telegram-service/conf"
	"mine.local/ocr-gallery/telegram-service/entity"
)

const COLLECTION_NAME = "telegram-user-account"

const INDEX_NAME = "telegram-id-uniq"

type UserAccountService interface {
	MapUserToAccount(ctx context.Context, userId int64) (uuid.UUID, error)
}

type UserAccountServiceImpl struct {
	mongoClient *mongo.Client
	collection  *mongo.Collection
}

// MapUserToAccount implements UserAccountService.
func (u *UserAccountServiceImpl) MapUserToAccount(ctx context.Context, userId int64) (uuid.UUID, error) {

	result := u.collection.FindOne(ctx, bson.D{{
		Key: "userid", Value: userId,
	}})

	binding := new(entity.MongoTgUserAccountBinding)

	if result.Err() == mongo.ErrNoDocuments {
		accountId, _ := uuid.NewRandom()
		binding.UserId = userId
		binding.AccountId = accountId
		_, err := u.collection.InsertOne(ctx, binding)
		return accountId, err
	}

	err := result.Decode(binding)
	if err != nil {
		return uuid.UUID{}, err
	}

	return binding.AccountId, nil
}

func NewUserAccountService(config *conf.MongodbConfig) (UserAccountService, error) {
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().
		ApplyURI(config.Uri).
		SetServerAPIOptions(serverAPI)
	// Create a new client and connect to the server
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, err
	}

	database := client.Database(config.Database)
	database.CreateCollection(
		context.Background(),
		COLLECTION_NAME,
	)

	collection := client.
		Database(config.Database).
		Collection(COLLECTION_NAME)

	collection.Indexes().CreateOne(
		context.Background(),
		mongo.IndexModel{
			Keys: bson.D{{Key: "userid", Value: -1}},
			Options: options.Index().
				SetUnique(true).
				SetName(INDEX_NAME),
		},
	)

	return &UserAccountServiceImpl{
		mongoClient: client,
		collection:  collection,
	}, err
}
