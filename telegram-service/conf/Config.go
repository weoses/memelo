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

type StorageServiceConfig struct {
	Uri string
}

type YoutubeServiceConfig struct {
	Uri string
}

type UserAccountConfig struct {
	StaticUuid string
}

type WebhookConfig struct {
	ExternalUrl string
}

type PermissionEntryConfig struct {
	AllowedUserIds []int64 `mapstructure:"AllowedUserIds"`
}

type PermissionsConfig struct {
	Create    *PermissionEntryConfig `mapstructure:"Create"`
	Delete    *PermissionEntryConfig `mapstructure:"Delete"`
	Recompute *PermissionEntryConfig `mapstructure:"Recompute"`
	Search    *PermissionEntryConfig `mapstructure:"Search"`
}

type Config struct {
	Server         *commonconfig.ServerConfig       `mapstructure:"server"`
	Log            *commonconfig.LoggingConfig      `mapstructure:"log"`
	Webhook        *WebhookConfig                   `mapstructure:"webhook"`
	Telegram       *TelegramConfig                  `mapstructure:"telegram"`
	Inline         *InlineConfig                    `mapstructure:"inline"`
	StorageService *StorageServiceConfig            `mapstructure:"storage-service"`
	YoutubeService *YoutubeServiceConfig            `mapstructure:"youtube-service"`
	UserAccount    *UserAccountConfig               `mapstructure:"user-account"`
	TempStorage    *commonconfig.MediaStorageConfig `mapstructure:"temp-storage"`
	Permissions    *PermissionsConfig               `mapstructure:"permissions"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}
	return cfg, nil
}
