package conf

import (
	"fmt"

	"github.com/spf13/viper"
	commonconfig "github.com/weoses/memelo/common/config"
)

type StorageServiceConfig struct {
	Uri string
}

type AccountConfig struct {
	Id string
}

type Config struct {
	Server         *commonconfig.ServerConfig  `mapstructure:"server"`
	Log            *commonconfig.LoggingConfig `mapstructure:"log"`
	StorageService *StorageServiceConfig       `mapstructure:"storage-service"`
	Account        *AccountConfig              `mapstructure:"account"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}
	return cfg, nil
}
