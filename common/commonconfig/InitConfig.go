package commonconfig

import (
	"fmt"

	"github.com/spf13/viper"
)

func InitConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$APPLICATION_CONFIGPATH")
	viper.AddConfigPath("$HOME/.appname")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
}

func NewServerConfig() *ServerConfig {
	conf := new(ServerConfig)
	viper.UnmarshalKey("server", conf)
	return conf
}

type ServerConfig struct {
	ListenAddress string
}
