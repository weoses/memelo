package main

import (
	"embed"
	"log"
	"log/slog"

	"github.com/weoses/memelo/common/config"
	"github.com/weoses/memelo/webapp/app"
	"github.com/weoses/memelo/webapp/conf"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

//go:embed frontend/dist
var frontendDist embed.FS

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
		fx.Supply(frontendDist),
		app.Module(),
	).Run()
}
