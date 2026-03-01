package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/weoses/memelo/common/config"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/ocr"
	"github.com/weoses/memelo/storage-service/service"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	config.InitConfig()
	loggingConfig := config.NewLoggingConfig()
	config.InitLogs(loggingConfig)

	fx.New(
		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.SlogLogger{Logger: slog.With()}
		}),
		fx.Provide(NewValidator),
		fx.Provide(config.NewServerConfig),

		fx.Provide(conf.NewImageEmbeddingConfig),
		fx.Provide(conf.NewImageConverterConfig),
		fx.Provide(conf.NewImageStorageConfig),
		fx.Provide(conf.NewMetadataStorageConfig),
		fx.Provide(conf.NewImageOcrConfig),

		fx.Provide(ocr.NewOcrProcessor),
		fx.Provide(ocr.NewImageConverter),
		fx.Provide(ocr.NewImageEmbeddingExtractor),

		fx.Provide(service.NewMetadataStorageService),
		fx.Provide(service.NewImageStorageService),
		fx.Provide(service.NewCreateImageServiceImpl),

		fx.Provide(
			fx.Annotate(
				service.NewSimpleSearcher,
				fx.ResultTags(`group:"searchers"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewIdSearcher,
				fx.ResultTags(`group:"searchers"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewFuzzySearcher,
				fx.ResultTags(`group:"searchers"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewAllSearcher,
				fx.ResultTags(`group:"searchers"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewMemeCrudService,
				fx.ParamTags(``, ``, ``, `group:"searchers"`),
			),
		),

		fx.Provide(service.NewSearchServiceApi),
		fx.Invoke(Startup),
	).Run()
}

func Startup(
	lc fx.Lifecycle,
	searchApi v1connect.SearchServiceHandler,
	cfg *config.ServerConfig,
) {
	mux := http.NewServeMux()
	path, handler := v1connect.NewSearchServiceHandler(searchApi)
	mux.Handle(path, handler)

	srv := &http.Server{
		Addr:    cfg.ListenAddress,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			go func() { _ = srv.Serve(ln) }()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
}

func NewValidator() *validator.Validate {
	return validator.New(validator.WithRequiredStructEnabled())
}
