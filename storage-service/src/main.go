package main

import (
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	oapiEcho "github.com/oapi-codegen/runtime/strictmiddleware/echo"
	"go.uber.org/fx"
	"mine.local/ocr-gallery/apispec/meme-storage/server"
	"mine.local/ocr-gallery/common/commonconfig"
	"mine.local/ocr-gallery/common/commonmiddleware"
	"mine.local/ocr-gallery/storage-service/conf"
	"mine.local/ocr-gallery/storage-service/service"
)

func main() {
	commonconfig.InitConfig()
	commonconfig.InitLogs()

	fx.New(
		fx.Provide(NewValidator),
		fx.Provide(commonconfig.NewServerConfig),
		fx.Provide(conf.NewOcrConfig),
		fx.Provide(conf.NewMetadataStorageConfig),
		fx.Provide(conf.NewImageStorageConfig),
		fx.Provide(service.NewMetadataStorageService),
		fx.Provide(service.NewImageStorageService),
		fx.Provide(service.NewOcrService),
		fx.Provide(service.NewApiHandler),
		fx.Invoke(Startup),
	).Run()
}

func Startup(
	storage server.StrictServerInterface,
	conf *commonconfig.ServerConfig,
) {
	srv := echo.New()

	server.RegisterHandlers(srv,
		server.NewStrictHandler(
			storage,
			[]oapiEcho.StrictEchoMiddlewareFunc{
				commonmiddleware.NewLoggingMiddleware(),
			}),
	)

	srv.Start(conf.ListenAddress)
}

func NewValidator() *validator.Validate {
	return validator.New(validator.WithRequiredStructEnabled())
}
