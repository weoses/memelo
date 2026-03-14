package service

import (
	"context"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/weoses/memelo/telegram-service/conf"
)

type TelegramBotService interface {
	StartBot(ctx context.Context)
}

type TelegramBotServiceImpl struct {
	inline  InlineHandlerService
	message MessageHandlerService
	bot     *tgbotapi.BotAPI
	log     *slog.Logger
}

func (srv *TelegramBotServiceImpl) StartBot(ctx context.Context) {
	srv.log.InfoContext(ctx, "Authorized", "account", srv.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := srv.bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			srv.bot.StopReceivingUpdates()
			return
		case update := <-updates:
			if update.InlineQuery != nil {
				srv.handleInlineRequest(ctx, &update)
			} else if update.Message != nil {
				srv.handleMessage(ctx, &update)
			} else if update.ChosenInlineResult != nil {
				srv.handleChosenResult(ctx, &update)
			}
		}
	}
}

func (srv *TelegramBotServiceImpl) handleMessage(ctx context.Context, update *tgbotapi.Update) {
	srv.log.InfoContext(ctx, "Bot message request")
	srv.log.DebugContext(ctx, "Bot message request details",
		"request", update.Message)

	answer, err := srv.message.ProcessMessage(ctx, update.Message)
	if err != nil {
		srv.log.ErrorContext(ctx, "Failed to process message", "error", err)
		message := tgbotapi.NewMessage(update.Message.Chat.ID, err.Error())
		message.ReplyToMessageID = update.Message.MessageID
		_, err = srv.bot.Send(message)
		if err != nil {
			srv.log.ErrorContext(ctx, "Failed to send message to bot", "error", err)
		}
		return
	}

	message := tgbotapi.NewMessage(update.Message.Chat.ID, answer.Message)
	message.ReplyToMessageID = update.Message.MessageID
	message.ParseMode = answer.ParseMode
	_, err = srv.bot.Send(message)
	if err != nil {
		srv.log.ErrorContext(ctx, "Failed to send message to bot", "error", err)
		return
	}
}

func (srv *TelegramBotServiceImpl) handleInlineRequest(ctx context.Context, update *tgbotapi.Update) {
	srv.log.InfoContext(ctx, "Bot inline request:",
		"query", update.InlineQuery.Query)

	srv.log.DebugContext(ctx, "Bot inline request details:",
		"data", update.InlineQuery)

	inlineResponse, err := srv.inline.ProcessQuery(ctx, update.InlineQuery)
	if err != nil {
		srv.log.ErrorContext(ctx, "failed to process inline query:", "error", err)
		return
	}
	srv.log.DebugContext(ctx, "Bot inline response details:",
		"data", inlineResponse)

	_, err = srv.bot.Request(inlineResponse)
	if err != nil {
		srv.log.ErrorContext(ctx, "Failed to send message to bot", "error", err)
		return
	}
}

func (srv *TelegramBotServiceImpl) handleChosenResult(ctx context.Context, u *tgbotapi.Update) {
	srv.log.InfoContext(ctx, "Bot choose result:",
		"query", u.ChosenInlineResult.Query,
		"resultId", u.ChosenInlineResult.ResultID)

	srv.log.DebugContext(ctx, "Bot chosen result details:",
		"data", u.ChosenInlineResult)

	err := srv.inline.ProcessChosenInlineQuery(ctx, u.ChosenInlineResult)
	if err != nil {
		srv.log.ErrorContext(ctx, "Failed to process chosen result", "error", err)
	}
}

func NewTelegramBot(config *conf.TelegramConfig) *tgbotapi.BotAPI {
	bot, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		slog.ErrorContext(context.Background(), "Bot api creation failed", "error", err)
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
		log:     slog.With("service", "TelegramBotService"),
	}
}
