package conf

import (
	"github.com/spf13/viper"
)

type TelegramConfig struct {
	Token string
	Debug bool
}

type InlineConfig struct {
	PageSize int
}

type MongodbConfig struct {
	Uri      string
	Database string
}

type StorageConfig struct {
	Uri string
}

type UserAccountConfig struct {
	StaticUuid string
}

func NewTelegramConfig() (*TelegramConfig, error) {
	conf := &TelegramConfig{}
	err := viper.UnmarshalKey("telegram", conf)
	return conf, err
}

func NewMongodbConfig() (*MongodbConfig, error) {
	conf := &MongodbConfig{}
	err := viper.UnmarshalKey("mongo", conf)
	return conf, err
}

func NewInlineConfig() (*InlineConfig, error) {
	conf := &InlineConfig{}
	err := viper.UnmarshalKey("inline", conf)
	return conf, err
}

func NewStorageConfig() (*StorageConfig, error) {
	conf := &StorageConfig{}
	err := viper.UnmarshalKey("storage-service", conf)
	return conf, err
}

func NewUserAccountConfig() (*UserAccountConfig, error) {
	conf := &UserAccountConfig{}
	err := viper.UnmarshalKey("user-account", conf)
	return conf, err
}
