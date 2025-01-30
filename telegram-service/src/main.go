package main

import (
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

	fx.New(
		fx.Provide(conf.NewTelegramConfig),
		fx.Provide(conf.NewMongodbConfig),
		fx.Provide(conf.NewStorageConfig),
		fx.Provide(conf.NewInlineConfig),
		fx.Provide(service.NewStorageConnector),
		fx.Provide(service.NewUserAccountService),
		fx.Provide(service.NewInlineService),
		fx.Provide(service.NewTelegramBotService),
		fx.Invoke(Statup),
	).Run()
}
