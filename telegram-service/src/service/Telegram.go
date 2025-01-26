package service

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"mine.local/ocr-gallery/telegram-service/conf"
)

type TelegramBotService interface {
	StartBot()
}

type TelegramBotServiceImpl struct {
	token string
	debug bool
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
			query := update.InlineQuery.Query

			log.Println(query)

			inlineConf := tgbotapi.InlineConfig{
				InlineQueryID: update.InlineQuery.ID,
				IsPersonal:    true,
				CacheTime:     0,
				Results: []interface{}{
					tgbotapi.NewInlineQueryResultArticle(
						"test1",
						"tittle",
						"body",
					),
				},
			}

			if _, err := bot.Request(inlineConf); err != nil {
				log.Println(err)
			}
		}
	}
}

func NewTelegramBotService(config *conf.TelegramConfig) TelegramBotService {
	return &TelegramBotServiceImpl{
		token: config.BotToken,
		debug: config.Debug,
	}
}
