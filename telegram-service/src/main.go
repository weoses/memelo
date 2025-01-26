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
		fx.Provide(service.NewTelegramBotService),
		fx.Invoke(Statup),
	).Run()
}
