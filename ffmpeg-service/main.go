package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"

	"connectrpc.com/connect"
	"github.com/weoses/memelo/common/config"
	"github.com/weoses/memelo/common/tracing"
	"github.com/weoses/memelo/ffmpeg-service/api"
	"github.com/weoses/memelo/ffmpeg-service/conf"
	"github.com/weoses/memelo/ffmpeg-service/service"
	ffmpegstorage "github.com/weoses/memelo/ffmpeg-service/storage"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func Startup(lc fx.Lifecycle, cfg *conf.Config, apiHandler *api.FfmpegServiceApi) {
	var srv *http.Server
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			mux := http.NewServeMux()
			mux.Handle(v1connect.NewFfmpegServiceHandler(apiHandler,
				connect.WithInterceptors(tracing.NewInterceptor(cfg.Log.ProjectId))))
			mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			srv = &http.Server{
				Addr:    cfg.Server.ListenAddress,
				Handler: h2c.NewHandler(mux, &http2.Server{}),
			}

			ln, err := net.Listen("tcp", cfg.Server.ListenAddress)
			if err != nil {
				return err
			}

			go func() { _ = srv.Serve(ln) }()
			slog.Info("ffmpeg-service started", "addr", cfg.Server.ListenAddress)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
}

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
		fx.Supply(cfg),
		fx.Provide(ffmpegstorage.NewTmpDataServiceS3Adapter),
		fx.Provide(ffmpegstorage.NewTmpDataService),
		fx.Provide(service.NewFfmpegJobService),
		fx.Provide(api.NewFfmpegServiceApi),
		fx.Invoke(Startup),
	).Run()
}
