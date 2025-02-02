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
	inline  InlineHandlerService
	message MessageHandlerService
	bot     *tgbotapi.BotAPI
}

func (srv *TelegramBotServiceImpl) StartBot() {
	log.Printf("Authorized on account %s", srv.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := srv.bot.GetUpdatesChan(u)

	for update := range updates {

		if update.InlineQuery != nil {
			srv.HandleInlineRequest(&update)
		} else if update.Message != nil {
			srv.HandleMessage(&update)
		}
	}
}

func (srv *TelegramBotServiceImpl) HandleMessage(update *tgbotapi.Update) {
	log.Printf("Bot message request: %+v", update.Message)
	answer, err := srv.message.ProcessMessage(update.Message)
	if err != nil {
		log.Println(err)
		message := tgbotapi.NewMessage(update.Message.Chat.ID, err.Error())
		message.ReplyToMessageID = update.Message.MessageID
		_, err = srv.bot.Send(message)
		return
	}

	message := tgbotapi.NewMessage(update.Message.Chat.ID, answer.Message)
	message.ReplyToMessageID = update.Message.MessageID
	_, err = srv.bot.Send(message)
	if err != nil {
		log.Println(err)
		return
	}
}
func (srv *TelegramBotServiceImpl) HandleInlineRequest(update *tgbotapi.Update) {
	log.Printf("Bot inline request: %+v", update.InlineQuery)
	inlineResponse, err := srv.inline.ProcessQuery(context.Background(), update.InlineQuery)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("Bot inline response: %+v", inlineResponse)

	_, err = srv.bot.Request(inlineResponse)
	if err != nil {
		log.Println(err)
		return
	}
}

func NewTelegramBot(config *conf.TelegramConfig) *tgbotapi.BotAPI {
	bot, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = config.Debug
	return bot
}

func NewTelegramBotService(
	bot *tgbotapi.BotAPI,
	inline InlineHandlerService,
	message MessageHandlerService,
) TelegramBotService {
	return &TelegramBotServiceImpl{
		bot:     bot,
		inline:  inline,
		message: message,
	}
}
