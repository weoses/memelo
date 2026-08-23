package main

import (
	"context"
	"log"
	"log/slog"

	"github.com/weoses/memelo/common/config"
	"github.com/weoses/memelo/common/tracing"
	"github.com/weoses/memelo/storage-service/app"
	"github.com/weoses/memelo/storage-service/conf"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func main() {
	config.InitConfig()
	cfg, err := conf.NewConfig()
	if err != nil {
		log.Fatal(err)
	}
	config.InitLogs(cfg.Log)

	shutdownTracer, err := tracing.InitTracer(context.Background(), "storage-service", cfg.Log.ProjectId)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = shutdownTracer(context.Background()) }()

	fx.New(
		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.SlogLogger{Logger: slog.With()}
		}),
		fx.Supply(cfg),
		app.Module(),
	).Run()
}
