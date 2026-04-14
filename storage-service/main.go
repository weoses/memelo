package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/weoses/memelo/common/config"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"github.com/weoses/memelo/storage-service/api"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/ocr"
	"github.com/weoses/memelo/storage-service/ocr/ffmpeg"
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
	cfg, err := conf.NewConfig()
	if err != nil {
		log.Fatal(err)
	}
	config.InitLogs(cfg.Log)

	fx.New(
		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.SlogLogger{Logger: slog.With()}
		}),
		fx.Provide(NewValidator),
		fx.Supply(cfg),
		fx.Provide(func(c *conf.Config) *conf.FfmpegConfig { return c.Ffmpeg }),

		fx.Provide(ocr.NewImageConverter),
		fx.Provide(gapi.NewImageEmbeddingExtractorGenai),
		fx.Provide(ffmpeg.NewVideo2Mp4Converter),
		fx.Provide(ffmpeg.NewVideo2FrameExtractor),
		fx.Provide(gapi.NewGeminiExtractor),

		fx.Provide(storage2.NewElasticTagStorage),
		fx.Provide(
			fx.Annotate(
				func(s storage2.ElasticTagStorage) storage2.ElasticMigrating {
					return s.(storage2.ElasticMigrating)
				},
				fx.ResultTags(`group:"migrators"`),
			),
		),
		fx.Provide(service.NewTagMetadataExtractService),
		fx.Provide(service.NewTagService),
		fx.Provide(api.NewTagsGrpcApi),

		fx.Provide(storage2.NewMetadataStorageService),
		fx.Provide(
			fx.Annotate(
				func(s storage2.MetadataStorageService) storage2.ElasticMigrating {
					return s.(storage2.ElasticMigrating)
				},
				fx.ResultTags(`group:"migrators"`),
			),
		),

		fx.Provide(storage2.NewMediaStorageServiceS3Adapter),
		fx.Provide(storage2.NewMediaStorageService),
		fx.Provide(storage2.NewTmpDataServiceS3Adapter),
		fx.Provide(storage2.NewTmpDataService),
		fx.Provide(service.NewExportService),
		fx.Provide(service.NewMemeCrudService),

		fx.Provide(service.NewRecomputeService),
		fx.Provide(api.NewRecomputeGrpcApi),

		// Pipeline steps (sorted by GetPos inside NewImageMetadataExtractService)
		fx.Provide(
			fx.Annotate(
				service.NewCalcHashPipelineStep,
				fx.ResultTags(`group:"pipeline_steps"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewImageCheckDuplicateByHashPipelineStep,
				fx.ResultTags(`group:"pipeline_steps"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewImageToJpegPipelineStep,
				fx.ResultTags(`group:"pipeline_steps"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewImageCalcEmbeddingPipelineStep,
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
				service.NewImgLlmExtractPipelineStep,
				fx.ResultTags(`group:"pipeline_steps"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewImageCreateThumbnailPipelineStep,
				fx.ResultTags(`group:"pipeline_steps"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewCalcTagsPipelineStep,
				fx.ResultTags(`group:"pipeline_steps"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewVidToMp4PipelineStep,
				fx.ResultTags(`group:"pipeline_steps"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewVidCalcEmbeddingsPipelineStep,
				fx.ResultTags(`group:"pipeline_steps"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewVidLlmExtractPipelineStep,
				fx.ResultTags(`group:"pipeline_steps"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				service.NewVidCreateThumbnailPipelineStep,
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
		fx.Provide(NewHealthCheck),
		fx.Invoke(
			fx.Annotate(
				storage2.RunMigrations,
				fx.ParamTags(`group:"migrators"`),
			),
		),
		fx.Invoke(Startup),
	).Run()
}

func Startup(
	lc fx.Lifecycle,
	searchApi v1connect.SearchServiceHandler,
	exportApi v1connect.ExportServiceHandler,
	tagsApi v1connect.TagsServiceHandler,
	recomputeApi v1connect.RecomputeServiceHandler,
	check *HealthCheck,
	cfg *conf.Config,
) {
	mux := http.NewServeMux()
	pathSearch, handlerSearch := v1connect.NewSearchServiceHandler(searchApi)
	pathExport, handlerExport := v1connect.NewExportServiceHandler(exportApi)
	pathTags, handlerTags := v1connect.NewTagsServiceHandler(tagsApi)
	pathRecompute, handlerRecompute := v1connect.NewRecomputeServiceHandler(recomputeApi)

	mux.Handle(pathSearch, handlerSearch)
	mux.Handle(pathExport, handlerExport)
	mux.Handle(pathTags, handlerTags)
	mux.Handle(pathRecompute, handlerRecompute)
	mux.Handle("/health", check)

	srv := &http.Server{
		Addr:         cfg.Server.ListenAddress,
		Handler:      h2c.NewHandler(mux, &http2.Server{}),
		WriteTimeout: time.Second * 300,
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
