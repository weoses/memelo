package conf

import (
	"fmt"

	"github.com/spf13/viper"
	commonconfig "github.com/weoses/memelo/common/config"
)

type YouTubeConfig struct {
	TempDir                string
	MaxConcurrentDownloads int
	JobTtlSeconds          int
	ApiKey                 string
	ApiHost                string
	VideoFormat            string
}

type Config struct {
	Server      *commonconfig.ServerConfig       `mapstructure:"server"`
	Log         *commonconfig.LoggingConfig      `mapstructure:"log"`
	TempStorage *commonconfig.MediaStorageConfig `mapstructure:"temp-storage"`
	YouTube     *YouTubeConfig                   `mapstructure:"youtube"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}
	return cfg, nil
}
