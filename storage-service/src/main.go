package main

import (
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	oapiEcho "github.com/oapi-codegen/runtime/strictmiddleware/echo"
	"github.com/weoses/memelo/common/commonconfig"
	"github.com/weoses/memelo/common/commonmiddleware"
	"github.com/weoses/memelo/gen/proto/memelo/v1/v1connect"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/service"
	"go.uber.org/fx"
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
		fx.Provide(commonmiddleware.NewLoggingMiddleware),
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
	middleware commonmiddleware.LoggingMiddlewareFunc,
) {
	srv := echo.New()
	srv.Debug = true
	v1connect.NewSearchServiceHandler()
	server.RegisterHandlers(srv,
		server.NewStrictHandler(
			storage,
			[]oapiEcho.StrictEchoMiddlewareFunc{
				oapiEcho.StrictEchoMiddlewareFunc(middleware),
			}),
	)

	srv.Start(conf.ListenAddress)
}

func NewValidator() *validator.Validate {
	return validator.New(validator.WithRequiredStructEnabled())
}
