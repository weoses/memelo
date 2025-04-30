package service

import (
	"context"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"mine.local/ocr-gallery/common/commonconst"
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
	slog.Info("Authorized", "account", srv.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := srv.bot.GetUpdatesChan(u)

	for update := range updates {

		if update.InlineQuery != nil {
			srv.HandleInlineRequest(&update)
		} else if update.Message != nil {
			srv.HandleMessage(&update)
		} else if update.ChosenInlineResult != nil {
			srv.HandleChosenResult(&update)
		}
	}
}

func (srv *TelegramBotServiceImpl) HandleMessage(update *tgbotapi.Update) {
	slog.Info("Bot message request")
	slog.Debug("Bot message request details",
		"request", update.Message)

	answer, err := srv.message.ProcessMessage(update.Message)
	if err != nil {
		slog.Error("Failed to process message", commonconst.ERR_LOG, err)
		message := tgbotapi.NewMessage(update.Message.Chat.ID, err.Error())
		message.ReplyToMessageID = update.Message.MessageID
		srv.bot.Send(message)
		return
	}

	message := tgbotapi.NewMessage(update.Message.Chat.ID, answer.Message)
	message.ReplyToMessageID = update.Message.MessageID
	message.ParseMode = answer.ParseMode
	_, err = srv.bot.Send(message)
	if err != nil {
		slog.Error("Failed to send message to bot", commonconst.ERR_LOG, err)
		return
	}
}
func (srv *TelegramBotServiceImpl) HandleInlineRequest(update *tgbotapi.Update) {
	slog.Info("Bot inline request:",
		"query", update.InlineQuery.Query)

	slog.Debug("Bot inline request details:",
		commonconst.DATA_LOG, update.InlineQuery)
	inlineResponse, err := srv.inline.ProcessQuery(context.Background(), update.InlineQuery)
	if err != nil {
		slog.Error("failed to process inline query:", commonconst.ERR_LOG, err)
		return
	}
	slog.Debug("Bot inline response details:",
		commonconst.DATA_LOG, inlineResponse)

	_, err = srv.bot.Request(inlineResponse)
	if err != nil {
		slog.Error("Failed to send message to bot", commonconst.ERR_LOG, err)
		return
	}
}

func (srv *TelegramBotServiceImpl) HandleChosenResult(u *tgbotapi.Update) {
	slog.Info("Bot choose result:",
		commonconst.QUERY_LOG, u.ChosenInlineResult.Query,
		"resultId", u.ChosenInlineResult.ResultID)

	slog.Debug("Bot chosen result details:",
		commonconst.DATA_LOG, u.ChosenInlineResult)

	err := srv.inline.ProcessChosenInlineQuery(context.Background(), u.ChosenInlineResult)
	if err != nil {
		slog.Error("Failed to process chosen result", commonconst.ERR_LOG, err)
	}

}

func NewTelegramBot(config *conf.TelegramConfig) *tgbotapi.BotAPI {
	bot, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		slog.Error("Bot api creation failed", commonconst.ERR_LOG, err)
		panic("bot api creation failed")
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
