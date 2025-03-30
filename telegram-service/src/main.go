package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/fx"
	"mine.local/ocr-gallery/common/commonconfig"
	"mine.local/ocr-gallery/telegram-service/conf"
	"mine.local/ocr-gallery/telegram-service/service"
)

func Statup(serv service.TelegramBotService) {
	serv.StartBot()
}

func main() {
	commonconfig.InitConfig()
	commonconfig.InitLogs()

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
