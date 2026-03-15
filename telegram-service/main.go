package main

import (
	"context"
	"log"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/weoses/memelo/common/config"
	"github.com/weoses/memelo/telegram-service/conf"
	"github.com/weoses/memelo/telegram-service/service"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func Startup(lc fx.Lifecycle, serv service.TelegramBotService) {
	ctx, cancel := context.WithCancel(context.Background())
	lc.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			go serv.StartBot(ctx)
			return nil
		},
		OnStop: func(stopCtx context.Context) error {
			cancel()
			return nil
		},
	})
}

func main() {
	config.InitConfig()
	loggingConfig, err := config.NewLoggingConfig()
	if err != nil {
		log.Fatal(err)
	}

	config.InitLogs(loggingConfig)

	fx.New(
		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.SlogLogger{Logger: slog.With()}
		}),
		fx.Provide(conf.NewTelegramConfig),
		fx.Provide(conf.NewUserAccountConfig),
		fx.Provide(conf.NewPostgresConfig),
		fx.Provide(conf.NewStorageConfig),
		fx.Provide(conf.NewInlineConfig),
		fx.Provide(service.NewTelegramBot),
		fx.Provide(service.NewStorageConnector),
		fx.Provide(fx.Annotate(service.NewTelegramFileResolverService, fx.From(new(*tgbotapi.BotAPI)))),
		fx.Provide(service.NewUserAccountService),
		fx.Provide(service.NewMessageHandlerService),
		fx.Provide(service.NewInlineService),
		fx.Provide(service.NewTelegramBotService),
		fx.Invoke(Startup),
	).Run()
}
