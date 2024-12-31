package main

import (
	"github.com/labstack/echo/v4"
	oapiEcho "github.com/oapi-codegen/runtime/strictmiddleware/echo"
	"go.uber.org/fx"
	"mine.local/ocr-gallery/image-collector/api/memesearch"
	"mine.local/ocr-gallery/image-collector/api/memestorage"
	"mine.local/ocr-gallery/image-collector/conf"
	"mine.local/ocr-gallery/image-collector/middleware"
	"mine.local/ocr-gallery/image-collector/service"
)

func main() {
	conf.InitConfig()

	fx.New(
		fx.Provide(conf.NewOcrConfig),
		fx.Provide(conf.NewMetadataStorageConfig),
		fx.Provide(conf.NewImageStorageConfig),
		fx.Provide(service.NewMetadataStorageService),
		fx.Provide(service.NewImageStorageService),
		fx.Provide(service.NewOcrService),
		fx.Provide(service.NewMemeStorageApiService),
		fx.Provide(service.NewMemeSearchApiService),
		fx.Invoke(Startup),
	).Run()
}

func Startup(
	memestorageApi memestorage.StrictServerInterface,
	memesearchApi memesearch.StrictServerInterface,
) {
	srv := echo.New()

	memestorage.RegisterHandlers(srv,
		memestorage.NewStrictHandler(
			memestorageApi,
			[]oapiEcho.StrictEchoMiddlewareFunc{
				middleware.NewLoggingMiddleware(),
			}),
	)

	memesearch.RegisterHandlers(
		srv,
		memesearch.NewStrictHandler(
			memesearchApi,
			[]oapiEcho.StrictEchoMiddlewareFunc{
				middleware.NewLoggingMiddleware(),
			}),
	)

	srv.Start(":7001")
}
