package conf

import (
	"fmt"

	"github.com/spf13/viper"
	commonconfig "github.com/weoses/memelo/common/config"
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

type WebhookConfig struct {
	ExternalUrl string
}

type Config struct {
	Server         *commonconfig.ServerConfig       `mapstructure:"server"`
	Log            *commonconfig.LoggingConfig      `mapstructure:"log"`
	Webhook        *WebhookConfig                   `mapstructure:"webhook"`
	Telegram       *TelegramConfig                  `mapstructure:"telegram"`
	Postgres       *PostgresConfig                  `mapstructure:"postgres"`
	Inline         *InlineConfig                    `mapstructure:"inline"`
	StorageService *StorageServiceConfig            `mapstructure:"storage-service"`
	UserAccount    *UserAccountConfig               `mapstructure:"user-account"`
	TempStorage    *commonconfig.MediaStorageConfig `mapstructure:"temp-storage"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}
	return cfg, nil
}
