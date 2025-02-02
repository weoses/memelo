package commonconfig

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func InitConfig() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

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

func NewServerConfig() *ServerConfig {
	conf := new(ServerConfig)
	viper.UnmarshalKey("server", conf)
	return conf
}

type ServerConfig struct {
	ListenAddress string
}
