package e2e_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
)

type Config struct {
	Uri      string `json:"uri"`
	Username string `json:"username"`
	Password string `json:"password"`
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
	_ = godotenv.Load(".env")
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

	tagsClient = v1connect.NewTagsServiceClient(
		http.DefaultClient,
		config.Uri,
		connect.WithInterceptors(NewAuthInterceptor(config.Username, config.Password)))
	searchClient = v1connect.NewSearchServiceClient(
		http.DefaultClient,
		config.Uri,
		connect.WithInterceptors(NewAuthInterceptor(config.Username, config.Password)))

	os.Exit(m.Run())
}

func genAuthData(username string, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}

type AuthInterceptor struct {
	Username string
	Password string
}

func (a AuthInterceptor) WrapUnary(unaryFunc connect.UnaryFunc) connect.UnaryFunc {
	f := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if a.Username != "" {
			req.Header().Add("Authorization", "basic "+genAuthData(a.Username, a.Password))
		}
		return unaryFunc(ctx, req)
	}
	return f
}

func (a AuthInterceptor) WrapStreamingClient(clientFunc connect.StreamingClientFunc) connect.StreamingClientFunc {
	panic("implement me")
}

func (a AuthInterceptor) WrapStreamingHandler(handlerFunc connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	//TODO implement me
	panic("implement me")
}

func NewAuthInterceptor(username string, password string) connect.Interceptor {
	return &AuthInterceptor{username, password}
}
