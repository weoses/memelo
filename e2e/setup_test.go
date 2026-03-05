package e2e_test

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
)

type Config struct {
	Uri string `json:"uri"`
}

var (
	tagsClient   v1connect.TagsServiceClient
	searchClient v1connect.SearchServiceClient
	config       Config
)

func genAccountId() string {
	accountIdUuid, _ := uuid.NewRandom()
	return accountIdUuid.String()
}

func TestMain(m *testing.M) {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to read config: %v\n", err)
		os.Exit(1)
	}

	err := viper.UnmarshalKey("storage-service", &config)
	if err != nil {
		panic(err)
	}

	if config.Uri == "" {
		fmt.Fprintln(os.Stderr, "storage-service.Uri is not set in config")
		os.Exit(1)
	}

	tagsClient = v1connect.NewTagsServiceClient(http.DefaultClient, config.Uri)
	searchClient = v1connect.NewSearchServiceClient(http.DefaultClient, config.Uri)

	os.Exit(m.Run())
}
