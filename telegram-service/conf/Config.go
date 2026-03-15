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

type PostgresConfig struct {
	DSN string
}

type StorageServiceConfig struct {
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

func NewPostgresConfig() (*PostgresConfig, error) {
	conf := &PostgresConfig{}
	err := viper.UnmarshalKey("postgres", conf)
	return conf, err
}

func NewInlineConfig() (*InlineConfig, error) {
	conf := &InlineConfig{}
	err := viper.UnmarshalKey("inline", conf)
	return conf, err
}

func NewStorageConfig() (*StorageServiceConfig, error) {
	conf := &StorageServiceConfig{}
	err := viper.UnmarshalKey("storage-service", conf)
	return conf, err
}

func NewUserAccountConfig() (*UserAccountConfig, error) {
	conf := &UserAccountConfig{}
	err := viper.UnmarshalKey("user-account", conf)
	return conf, err
}
