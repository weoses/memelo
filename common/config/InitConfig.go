package config

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

func InitConfig() {
	_ = godotenv.Load(".env")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$APPLICATION_CONFIGPATH")
	viper.AddConfigPath("$HOME/.appname")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	// workaround because viper does not treat env vars the same as other config
	for _, key := range viper.AllKeys() {
		val := viper.Get(key)
		viper.Set(key, val)
	}

}

func NewServerConfig() (*ServerConfig, error) {
	conf := new(ServerConfig)
	err := viper.UnmarshalKey("server", conf)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return conf, nil
}

func NewLoggingConfig() (*LoggingConfig, error) {
	conf := new(LoggingConfig)
	err := viper.UnmarshalKey("log", conf)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return conf, nil
}

type ServerConfig struct {
	ListenAddress string
}

type LoggingConfig struct {
	Level string
}

type MediaStorageConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Secure    bool
}
