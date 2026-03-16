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
				if update.Message.IsCommand() {
					srv.handleCommand(ctx, update.Message)
				} else {
					srv.handleMessage(ctx, update.Message)
				}
			} else if update.ChosenInlineResult != nil {
				srv.handleChosenResult(ctx, &update)
			}
		}
	}
}

func (srv *TelegramBotServiceImpl) handleCommand(ctx context.Context, requestMessage *tgbotapi.Message) {
	srv.log.InfoContext(ctx, "Bot message request")
	srv.log.DebugContext(ctx, "Bot message request details",
		"request", requestMessage)

	if requestMessage.Command() == "addtag" {
		responseData, err := srv.message.ProcessCommandAddTag(ctx, requestMessage)
		if err != nil {
			srv.sendCommonErrorMessage(ctx, requestMessage, err)
		}

		err = srv.sendCommonResponseMessage(ctx, requestMessage, responseData)
		if err != nil {
			srv.log.ErrorContext(ctx, "Failed to send message to bot", "error", err)
			srv.sendCommonErrorMessage(ctx, requestMessage, err)
			return
		}

	}
}

func (srv *TelegramBotServiceImpl) handleMessage(ctx context.Context, requestMessage *tgbotapi.Message) {
	srv.log.InfoContext(ctx, "Bot message request")
	srv.log.DebugContext(ctx, "Bot message request details",
		"request", requestMessage)

	answer, err := srv.message.ProcessImageMessage(ctx, requestMessage)
	if err != nil {
		srv.log.ErrorContext(ctx, "Failed to process message", "error", err)
		srv.sendCommonErrorMessage(ctx, requestMessage, err)
		return
	}

	err = srv.sendCommonResponseMessage(ctx, requestMessage, answer)
	if err != nil {
		srv.log.ErrorContext(ctx, "Failed to send message to bot", "error", err)
		srv.sendCommonErrorMessage(ctx, requestMessage, err)
		return
	}
}

func (srv *TelegramBotServiceImpl) sendCommonResponseMessage(ctx context.Context, requestMessage *tgbotapi.Message, answer *MessageHandlerResponse) error {
	responseMessage := tgbotapi.NewMessage(requestMessage.Chat.ID, answer.Message)
	responseMessage.ReplyToMessageID = requestMessage.MessageID
	responseMessage.ParseMode = answer.ParseMode
	_, err := srv.bot.Send(responseMessage)
	return err
}

func (srv *TelegramBotServiceImpl) sendCommonErrorMessage(ctx context.Context, requestMessage *tgbotapi.Message, err error) {
	errorResponseMessage := tgbotapi.NewMessage(requestMessage.Chat.ID, err.Error())
	errorResponseMessage.ReplyToMessageID = requestMessage.MessageID
	_, err = srv.bot.Send(errorResponseMessage)
	if err != nil {
		srv.log.ErrorContext(ctx, "Failed to send message to bot", "error", err)
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
