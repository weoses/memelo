package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/weoses/memelo/telegram-service/conf"
)

type TelegramBotService interface {
	Handler() http.Handler
	RegisterWebhook() error
	RemoveWebhook() error
}

type TelegramBotServiceImpl struct {
	inline     InlineHandlerService
	message    MessageHandlerService
	bot        *tgbotapi.BotAPI
	webhookCfg *conf.WebhookConfig
	log        *slog.Logger
	cancel     context.CancelFunc
}

func (s *TelegramBotServiceImpl) Handler() http.Handler {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	updates := make(chan tgbotapi.Update, 100)
	go s.dispatchUpdates(ctx, updates)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var update tgbotapi.Update
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		updates <- update
	})
}

func (s *TelegramBotServiceImpl) RegisterWebhook() error {
	wh, err := tgbotapi.NewWebhook(s.webhookCfg.ExternalUrl)
	if err != nil {
		return err
	}
	_, err = s.bot.Request(wh)
	return err
}

func (s *TelegramBotServiceImpl) RemoveWebhook() error {
	if s.cancel != nil {
		s.cancel()
	}
	_, err := s.bot.Request(tgbotapi.DeleteWebhookConfig{})
	return err
}

func (s *TelegramBotServiceImpl) dispatchUpdates(ctx context.Context, updates <-chan tgbotapi.Update) {
	s.log.InfoContext(ctx, "Authorized", "account", s.bot.Self.UserName)
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-updates:
			if update.InlineQuery != nil {
				s.handleInlineRequest(ctx, &update)
			} else if update.Message != nil {
				if update.Message.IsCommand() {
					s.handleCommand(ctx, update.Message)
				} else {
					s.handleMessage(ctx, update.Message)
				}
			} else if update.ChosenInlineResult != nil {
				s.handleChosenResult(ctx, &update)
			}
		}
	}
}

func (s *TelegramBotServiceImpl) handleCommand(ctx context.Context, requestMessage *tgbotapi.Message) {
	s.log.InfoContext(ctx, "Bot message request")
	s.log.DebugContext(ctx, "Bot message request details",
		"request", requestMessage)

	if requestMessage.Command() == "addtag" {
		responseData, err := s.message.ProcessCommandAddTag(ctx, requestMessage)
		if err != nil {
			s.sendCommonErrorMessage(ctx, requestMessage, err)
		}

		err = s.sendCommonResponseMessage(ctx, requestMessage, responseData)
		if err != nil {
			s.log.ErrorContext(ctx, "Failed to send message to bot", "error", err)
			s.sendCommonErrorMessage(ctx, requestMessage, err)
			return
		}

	}
}

func (s *TelegramBotServiceImpl) handleMessage(ctx context.Context, requestMessage *tgbotapi.Message) {
	s.log.InfoContext(ctx, "Bot message request")
	s.log.DebugContext(ctx, "Bot message request details",
		"request", requestMessage)

	var answer *MessageHandlerResponse
	var err error
	if requestMessage.Video != nil {
		answer, err = s.message.ProcessVideoMessage(ctx, requestMessage)
	} else if requestMessage.Photo != nil {
		answer, err = s.message.ProcessImageMessage(ctx, requestMessage)
	} else {
		err = errors.New("message dont contain any media temp")
	}

	if err != nil {
		s.log.ErrorContext(ctx, "Failed to process message", "error", err)
		s.sendCommonErrorMessage(ctx, requestMessage, err)
		return
	}

	err = s.sendCommonResponseMessage(ctx, requestMessage, answer)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to send message to bot", "error", err)
		s.sendCommonErrorMessage(ctx, requestMessage, err)
		return
	}
}

func (s *TelegramBotServiceImpl) sendCommonResponseMessage(ctx context.Context, requestMessage *tgbotapi.Message, answer *MessageHandlerResponse) error {
	responseMessage := tgbotapi.NewMessage(requestMessage.Chat.ID, answer.Message)
	responseMessage.ReplyToMessageID = requestMessage.MessageID
	responseMessage.ParseMode = answer.ParseMode
	_, err := s.bot.Send(responseMessage)
	return err
}

func (s *TelegramBotServiceImpl) sendCommonErrorMessage(ctx context.Context, requestMessage *tgbotapi.Message, err error) {
	errorResponseMessage := tgbotapi.NewMessage(requestMessage.Chat.ID, err.Error())
	errorResponseMessage.ReplyToMessageID = requestMessage.MessageID
	_, err = s.bot.Send(errorResponseMessage)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to send message to bot", "error", err)
	}
}

func (s *TelegramBotServiceImpl) handleInlineRequest(ctx context.Context, update *tgbotapi.Update) {
	s.log.InfoContext(ctx, "Bot inline request:",
		"query", update.InlineQuery.Query)

	s.log.DebugContext(ctx, "Bot inline request details:",
		"temp", update.InlineQuery)

	inlineResponse, err := s.inline.ProcessQuery(ctx, update.InlineQuery)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to process inline query:", "error", err)
		return
	}
	s.log.DebugContext(ctx, "Bot inline response details:",
		"temp", inlineResponse)

	_, err = s.bot.Request(inlineResponse)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to send message to bot", "error", err)
		return
	}
}

func (s *TelegramBotServiceImpl) handleChosenResult(ctx context.Context, u *tgbotapi.Update) {
	s.log.InfoContext(ctx, "Bot choose result:",
		"query", u.ChosenInlineResult.Query,
		"resultId", u.ChosenInlineResult.ResultID)

	s.log.DebugContext(ctx, "Bot chosen result details:",
		"temp", u.ChosenInlineResult)

	err := s.inline.ProcessChosenInlineQuery(ctx, u.ChosenInlineResult)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to process chosen result", "error", err)
	}
}

func NewTelegramBot(config *conf.Config) *tgbotapi.BotAPI {
	bot, err := tgbotapi.NewBotAPI(config.Telegram.Token)
	if err != nil {
		slog.ErrorContext(context.Background(), "Bot api creation failed", "error", err)
		panic("bot api creation failed")
	}
	bot.Debug = config.Telegram.Debug
	return bot
}

func NewTelegramBotService(
	bot *tgbotapi.BotAPI,
	inline InlineHandlerService,
	message MessageHandlerService,
	webhookCfg *conf.WebhookConfig,
) TelegramBotService {
	return &TelegramBotServiceImpl{
		bot:        bot,
		inline:     inline,
		message:    message,
		webhookCfg: webhookCfg,
		log:        slog.With("service", "TelegramBotService"),
	}
}
