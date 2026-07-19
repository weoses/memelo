package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/weoses/memelo/common/config"
	"github.com/weoses/memelo/telegram-service/conf"
	"github.com/weoses/memelo/telegram-service/service"
	tgstorage "github.com/weoses/memelo/telegram-service/storage"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func Startup(lc fx.Lifecycle, cfg *conf.Config, svc service.TelegramBotService) {
	var srv *http.Server
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := svc.RegisterWebhook(); err != nil {
				return err
			}
			mux := http.NewServeMux()
			mux.Handle("/webhook", svc.Handler())
			mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
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
			_ = svc.RemoveWebhook()
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
		fx.Provide(service.NewTelegramBot),
		fx.Provide(tgstorage.NewTmpDataServiceS3Adapter),
		fx.Provide(tgstorage.NewTmpDataService),
		fx.Provide(service.NewStorageConnector),
		fx.Provide(service.NewYouTubeConnector),
		fx.Provide(fx.Annotate(service.NewTelegramFileResolverService, fx.From(new(*tgbotapi.BotAPI)))),
		fx.Provide(service.NewPermissionService),
		fx.Provide(service.NewMessageHandlerService),
		fx.Provide(service.NewQueryProcessorFactory),
		fx.Provide(service.NewInlineService),
		fx.Provide(service.NewTelegramBotService),
		fx.Invoke(Startup),
	).Run()
}
