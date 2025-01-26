package conf

import (
	"github.com/spf13/viper"
)

type TelegramConfig struct {
	BotToken string
	Debug    bool
}

type AccountsConfig struct {
}

func NewTelegramConfig() (*TelegramConfig, error) {
	conf := &TelegramConfig{}
	err := viper.UnmarshalKey("telegram", conf)
	return conf, err
}
