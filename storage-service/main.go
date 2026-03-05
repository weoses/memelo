package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/weoses/memelo/common/config"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"github.com/weoses/memelo/storage-service/api"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/ocr"
	"github.com/weoses/memelo/storage-service/ocr/gapi"
	"github.com/weoses/memelo/storage-service/service"
	storage2 "github.com/weoses/memelo/storage-service/storage"
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
		fx.Provide(conf.NewElasticTagConfig),

		fx.Provide(gapi.NewOcrProcessor),
		fx.Provide(ocr.NewImageConverter),
		fx.Provide(gapi.NewImageEmbeddingExtractor),

		fx.Provide(storage2.NewElasticTagStorage),
		fx.Provide(service.NewTagMetadataExtractService),
		fx.Provide(service.NewTagService),
		fx.Provide(api.NewTagsGrpcApi),

		fx.Provide(storage2.NewMetadataStorageService),
		fx.Provide(storage2.NewImageStorageService),
		fx.Provide(service.NewExportService),
		fx.Provide(service.NewMemeCrudService),

		// Pipeline steps (sorted by GetPos inside NewImageMetadataExtractService)
		fx.Provide(
			fx.Annotate(
				service.NewCalcHashPipelineStep,
				fx.ResultTags(`group:"pipeline_steps"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewCheckDuplicateByHashPipelineStep,
				fx.ResultTags(`group:"pipeline_steps"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewToJpegPipelineStep,
				fx.ResultTags(`group:"pipeline_steps"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewCalcEmbeddingPipelineStep,
				fx.ResultTags(`group:"pipeline_steps"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewCheckDuplicateByEmbeddingPipelineStep,
				fx.ResultTags(`group:"pipeline_steps"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewOcrImagePipelineStep,
				fx.ResultTags(`group:"pipeline_steps"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewCreateThumbnailPipelineStep,
				fx.ResultTags(`group:"pipeline_steps"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewCalcSizesPipelineStep,
				fx.ResultTags(`group:"pipeline_steps"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewImageMetadataExtractService,
				fx.ParamTags(`group:"pipeline_steps"`),
			),
		),

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
				service.NewTextEmbeddingSearcher,
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
				service.NewSearchServiceImpl,
				fx.ParamTags(`group:"searchers"`, ``),
			),
		),

		fx.Provide(api.NewSearchServiceApi),
		fx.Provide(api.NewExportServiceApi),
		fx.Invoke(Startup),
	).Run()
}

func Startup(
	lc fx.Lifecycle,
	searchApi v1connect.SearchServiceHandler,
	exportApi v1connect.ExportServiceHandler,
	tagsApi v1connect.TagsServiceHandler,
	cfg *config.ServerConfig,
) {
	mux := http.NewServeMux()
	pathSearch, handlerSearch := v1connect.NewSearchServiceHandler(searchApi)
	pathExport, handlerExport := v1connect.NewExportServiceHandler(exportApi)
	pathTags, handlerTags := v1connect.NewTagsServiceHandler(tagsApi)

	mux.Handle(pathSearch, handlerSearch)
	mux.Handle(pathExport, handlerExport)
	mux.Handle(pathTags, handlerTags)

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
