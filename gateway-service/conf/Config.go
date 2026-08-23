package conf

import (
	"fmt"

	"github.com/spf13/viper"
	commonconfig "github.com/weoses/memelo/common/config"
)

// ProxyTargetConfig describes one backend the gateway reverse-proxies to.
type ProxyTargetConfig struct {
	Uri                  string
	RequireGoogleIDToken bool
}

type BasicAuthConfig struct {
	Username string
	Password string
}

type Config struct {
	Server          *commonconfig.ServerConfig  `mapstructure:"server"`
	Log             *commonconfig.LoggingConfig `mapstructure:"log"`
	TelegramService *ProxyTargetConfig          `mapstructure:"telegram-service"`
	WebappService   *ProxyTargetConfig          `mapstructure:"webapp-service"`
	BasicAuth       *BasicAuthConfig            `mapstructure:"basic-auth"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}
	return cfg, nil
}
