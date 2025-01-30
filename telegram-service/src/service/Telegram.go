package service

import (
	"context"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"mine.local/ocr-gallery/telegram-service/conf"
)

type TelegramBotService interface {
	StartBot()
}

type TelegramBotServiceImpl struct {
	token  string
	debug  bool
	inline InlineService
}

func (srv *TelegramBotServiceImpl) StartBot() {
	bot, err := tgbotapi.NewBotAPI(srv.token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = srv.debug

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.InlineQuery != nil {
			log.Printf("Bot inline request: %+v", update.InlineQuery)
			inlineResponse, err := srv.inline.ProcessQuery(context.Background(), update.InlineQuery)
			if err != nil {
				log.Println(err)
				continue
			}
			log.Printf("Bot inline response: %+v", inlineResponse)

			_, err = bot.Request(inlineResponse)
			if err != nil {
				log.Println(err)
				continue
			}
		}
	}
}

func NewTelegramBotService(
	config *conf.TelegramConfig,
	inline InlineService,
) TelegramBotService {
	return &TelegramBotServiceImpl{
		token:  config.BotToken,
		debug:  config.Debug,
		inline: inline,
	}
}
