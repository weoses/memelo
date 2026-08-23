package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"

	"github.com/weoses/memelo/common/config"
	"github.com/weoses/memelo/common/tracing"
	"github.com/weoses/memelo/gateway-service/conf"
	"github.com/weoses/memelo/gateway-service/proxy"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func Startup(lc fx.Lifecycle, cfg *conf.Config) {
	var srv *http.Server
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			mux, err := proxy.NewMux(ctx, cfg)
			if err != nil {
				return err
			}
			srv = &http.Server{
				Addr:    cfg.Server.ListenAddress,
				Handler: mux,
			}
			ln, err := net.Listen("tcp", cfg.Server.ListenAddress)
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

func main() {
	config.InitConfig()
	cfg, err := conf.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	config.InitLogs(cfg.Log)
	slog.Info("DEBUG cfg dump", "basicAuth", cfg.BasicAuth, "telegram", cfg.TelegramService, "webapp", cfg.WebappService)

	shutdownTracer, err := tracing.InitTracer(context.Background(), "gateway-service", cfg.Log.ProjectId)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = shutdownTracer(context.Background()) }()

	fx.New(
		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.SlogLogger{Logger: slog.With()}
		}),
		fx.Supply(cfg),
		fx.Invoke(Startup),
	).Run()
}
