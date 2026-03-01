package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/weoses/memelo/common/config"
	"github.com/weoses/memelo/telegram-service/conf"
	"github.com/weoses/memelo/telegram-service/service"
	"go.uber.org/fx"
)

func Statup(serv service.TelegramBotService) {
	serv.StartBot()
}

func main() {
	config.InitConfig()
	config.InitLogs()

	fx.New(
		fx.Provide(conf.NewTelegramConfig),
		fx.Provide(conf.NewUserAccountConfig),
		fx.Provide(conf.NewMongodbConfig),
		fx.Provide(conf.NewStorageConfig),
		fx.Provide(conf.NewInlineConfig),
		fx.Provide(service.NewTelegramBot),
		fx.Provide(service.NewStorageConnector),
		fx.Provide(fx.Annotate(service.NewTelegramFileResolverService, fx.From(new(*tgbotapi.BotAPI)))),
		fx.Provide(service.NewUserAccountService),
		fx.Provide(service.NewMessageHandlerService),
		fx.Provide(service.NewInlineService),
		fx.Provide(service.NewTelegramBotService),
		fx.Invoke(Statup),
	).Run()
}
