package main

import (
	"github.com/labstack/echo/v4"
	oapiEcho "github.com/oapi-codegen/runtime/strictmiddleware/echo"
	"go.uber.org/fx"
	"mine.local/ocr-gallery/apispec/ocr-server/server"
	"mine.local/ocr-gallery/common/commonconfig"
	"mine.local/ocr-gallery/common/commonmiddleware"
	"mine.local/ocr-gallery/ocr-server/api"
	"mine.local/ocr-gallery/ocr-server/conf"
	"mine.local/ocr-gallery/ocr-server/service"
)

func main() {
	commonconfig.InitConfig()

	fx.New(
		fx.Provide(commonconfig.NewServerConfig),
		fx.Provide(conf.NewImageConverterConfig),
		fx.Provide(conf.NewImageEmbeddingConfig),
		fx.Provide(api.NewApiHandler),
		fx.Provide(service.NewImageEmbeddingExtractor),
		fx.Provide(service.NewVisionImageClient),
		fx.Provide(service.NewOcrProcessor),
		fx.Provide(service.NewImageConverter),
		fx.Provide(service.NewImageService),
		fx.Invoke(Startup),
	).Run()
}

func Startup(handler server.StrictServerInterface, config *commonconfig.ServerConfig) {
	srv := echo.New()

	server.RegisterHandlers(
		srv,
		server.NewStrictHandler(
			handler,
			[]oapiEcho.StrictEchoMiddlewareFunc{
				commonmiddleware.NewLoggingMiddleware(),
			}),
	)

	srv.Start(config.ListenAddress)
}
