package app

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/go-playground/validator/v10"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"github.com/weoses/memelo/storage-service/api"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/middleware"
	"github.com/weoses/memelo/storage-service/ocr"
	"github.com/weoses/memelo/storage-service/ocr/ffmpeg"
	"github.com/weoses/memelo/storage-service/ocr/gapi"
	openrouterpkg "github.com/weoses/memelo/storage-service/ocr/openrouter"
	"github.com/weoses/memelo/storage-service/service"
	storage2 "github.com/weoses/memelo/storage-service/storage"
	"go.uber.org/fx"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Module returns the complete fx options for the storage service.
//
// Callers must additionally provide:
//   - net.Listener (the pre-bound TCP listener)
//
// LLM providers (ocr.LlmMediaExtractor and the raw_embedder ocr.LlmEmbeddingExtractor)
// default to the real Gemini implementations and can be replaced in tests via fx.Decorate.
func Module() fx.Option {
	return fx.Options(
		fx.Provide(func() *validator.Validate { return validator.New(validator.WithRequiredStructEnabled()) }),
		fx.Provide(func(c *conf.Config) *conf.FfmpegConfig { return c.Ffmpeg }),
		fx.Provide(func(c *conf.Config) *conf.CommonExtractingConfig { return c.Extracting }),

		fx.Provide(ocr.NewImageConverter),
		// LLM provider implementations — selected by extractor-provider / embedder-provider config.
		// Override with fx.Decorate in tests.
		fx.Provide(func(cfg *conf.Config) (ocr.LlmMediaExtractor, error) {
			if cfg.ExtractorProvider == "openrouter" {
				return openrouterpkg.NewOpenRouterExtractor(cfg)
			}
			return gapi.NewGeminiExtractor(cfg)
		}),
		fx.Provide(
			fx.Annotate(
				func(cfg *conf.Config, fe ocr.Video2FrameExtractor) (ocr.LlmEmbeddingExtractor, error) {
					if cfg.EmbedderProvider == "openrouter" {
						return openrouterpkg.NewOpenRouterEmbedder(cfg, fe)
					}
					return gapi.NewImageEmbeddingExtractorGenai(cfg)
				},
				fx.ResultTags(`name:"raw_embedder"`),
			),
		),
		fx.Provide(
			fx.Annotate(
				ocr.NewCachedEmbeddingExtractor,
				fx.ParamTags(`name:"raw_embedder"`),
			),
		),
		fx.Provide(ffmpeg.NewVideo2Mp4Converter),
		fx.Provide(ffmpeg.NewVideo2FrameExtractor),
		fx.Provide(ffmpeg.NewVideoSlicer),

		fx.Provide(storage2.NewElasticTagStorage),
		fx.Provide(
			fx.Annotate(
				func(s storage2.ElasticTagStorage) storage2.ElasticMigrating { return s.(storage2.ElasticMigrating) },
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
		fx.Provide(service.NewMediaEncoderService),
		fx.Provide(service.NewPipelineSaveService),
		fx.Provide(service.NewMemeCrudService),

		fx.Provide(service.NewInMemoryRecomputeJobStorage),
		fx.Provide(service.NewRecomputeService),
		fx.Provide(api.NewRecomputeGrpcApi),

		fx.Provide(fx.Annotate(service.NewCalcHashPipelineStep, fx.ResultTags(`group:"pipeline_steps"`))),
		fx.Provide(fx.Annotate(service.NewImageCheckDuplicateByHashPipelineStep, fx.ResultTags(`group:"pipeline_steps"`))),
		fx.Provide(fx.Annotate(service.NewEncodeOriginalPipelineStep, fx.ResultTags(`group:"pipeline_steps"`))),
		fx.Provide(fx.Annotate(service.NewImageCalcEmbeddingPipelineStep, fx.ResultTags(`group:"pipeline_steps"`))),
		fx.Provide(fx.Annotate(service.NewCheckDuplicateByEmbeddingPipelineStep, fx.ResultTags(`group:"pipeline_steps"`))),
		fx.Provide(fx.Annotate(service.NewImgLlmExtractPipelineStep, fx.ResultTags(`group:"pipeline_steps"`))),
		fx.Provide(fx.Annotate(service.NewCreateThumbnailPipelineStep, fx.ResultTags(`group:"pipeline_steps"`))),
		fx.Provide(fx.Annotate(service.NewCalcTagsPipelineStep, fx.ResultTags(`group:"pipeline_steps"`))),
		fx.Provide(fx.Annotate(service.NewVidSlicePipelineStep, fx.ResultTags(`group:"pipeline_steps"`))),
		fx.Provide(fx.Annotate(service.NewVidCalcEmbeddingsPipelineStep, fx.ResultTags(`group:"pipeline_steps"`))),
		fx.Provide(fx.Annotate(service.NewVidLlmExtractPipelineStep, fx.ResultTags(`group:"pipeline_steps"`))),
		fx.Provide(fx.Annotate(service.NewImageMetadataExtractService, fx.ParamTags(`group:"pipeline_steps"`))),

		fx.Provide(fx.Annotate(service.NewIdSearcher, fx.ResultTags(`group:"searchers"`))),
		fx.Provide(fx.Annotate(service.NewHybridSearcher, fx.ResultTags(`group:"searchers"`))),
		fx.Provide(fx.Annotate(service.NewAllSearcher, fx.ResultTags(`group:"searchers"`))),
		fx.Provide(fx.Annotate(service.NewSearchServiceImpl, fx.ParamTags(`group:"searchers"`, ``))),

		fx.Provide(api.NewSearchServiceApi),
		fx.Provide(api.NewExportServiceApi),

		fx.Provide(func(c *conf.Config) (net.Listener, error) {
			return net.Listen("tcp", c.Server.ListenAddress)
		}),

		fx.Invoke(fx.Annotate(storage2.RunMigrations, fx.ParamTags(`group:"migrators"`))),
		fx.Invoke(startup),
	)
}

// startup wires the Connect-RPC handlers onto the pre-bound net.Listener and
// registers start/stop lifecycle hooks for the HTTP server.
func startup(
	lc fx.Lifecycle,
	ln net.Listener,
	searchApi v1connect.SearchServiceHandler,
	exportApi v1connect.ExportServiceHandler,
	tagsApi v1connect.TagsServiceHandler,
	recomputeApi v1connect.RecomputeServiceHandler,
) {
	interceptors := connect.WithInterceptors(middleware.NewLoggingInterceptor(slog.With("service", "router")))

	mux := http.NewServeMux()
	mux.Handle(v1connect.NewSearchServiceHandler(searchApi, interceptors))
	mux.Handle(v1connect.NewExportServiceHandler(exportApi, interceptors))
	mux.Handle(v1connect.NewTagsServiceHandler(tagsApi, interceptors))
	mux.Handle(v1connect.NewRecomputeServiceHandler(recomputeApi, interceptors))
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{
		Handler:      h2c.NewHandler(mux, &http2.Server{}),
		WriteTimeout: time.Second * 300,
	}

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() { _ = srv.Serve(ln) }()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
}
