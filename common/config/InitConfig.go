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

type ServerConfig struct {
	ListenAddress string
}

type LoggingConfig struct {
	Level string
	// Format selects the slog handler: "text" (default, human-readable) or
	// "json" (Cloud Logging structured-log conventions -- severity/message
	// keys, so Log Explorer's summary line and severity filter work).
	Format string
	// ProjectId enables Cloud Trace correlation ("show entries for this
	// trace" in Cloud Run's request log) by letting log lines carry a
	// fully-qualified logging.googleapis.com/trace attribute. Leave empty
	// (the local/docker-compose default) to skip trace correlation
	// entirely -- Terraform sets this to the real project id in Cloud Run.
	ProjectId string
}

type MediaStorageConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Secure    bool
}
