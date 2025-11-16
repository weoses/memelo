package main

import (
	"github.com/labstack/echo/v4"
	oapiEcho "github.com/oapi-codegen/runtime/strictmiddleware/echo"
	"github.com/weoses/memelo/apispec/ocr-server/server"
	"github.com/weoses/memelo/common/commonconfig"
	"github.com/weoses/memelo/common/commonmiddleware"
	"github.com/weoses/memelo/ocr-server/api"
	"github.com/weoses/memelo/ocr-server/conf"
	"github.com/weoses/memelo/ocr-server/service"
	"go.uber.org/fx"
)

func main() {
	commonconfig.InitConfig()
	commonconfig.InitLogs()

	fx.New(
		fx.Provide(commonconfig.NewServerConfig),
		fx.Provide(conf.NewImageConverterConfig),
		fx.Provide(conf.NewImageEmbeddingConfig),
		fx.Provide(api.NewApiHandler),
		fx.Provide(commonmiddleware.NewLoggingMiddleware),
		fx.Provide(service.NewImageEmbeddingExtractor),
		fx.Provide(service.NewVisionImageClient),
		fx.Provide(service.NewOcrProcessor),
		fx.Provide(service.NewImageConverter),
		fx.Provide(service.NewImageService),
		fx.Invoke(Startup),
	).Run()
}

func Startup(handler server.StrictServerInterface,
	config *commonconfig.ServerConfig,
	loggingMiddleware commonmiddleware.LoggingMiddlewareFunc) {
	srv := echo.New()
	srv.Debug = true

	server.RegisterHandlers(
		srv,
		server.NewStrictHandler(
			handler,
			[]oapiEcho.StrictEchoMiddlewareFunc{
				oapiEcho.StrictEchoMiddlewareFunc(loggingMiddleware),
			}),
	)
	err := srv.Start(config.ListenAddress)
	if err != nil {
		panic(err)
	}
}
