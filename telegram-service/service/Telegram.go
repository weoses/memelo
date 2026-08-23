package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/tracing"
	"github.com/weoses/memelo/telegram-service/conf"
	"go.opentelemetry.io/otel/trace"
)

const (
	msgProcessing = "⏳ Processing..."
	msgStartIntro = "Welcome to Memelo Bot!\n\n" +
		"Send me an image or video to save it as a meme.\n\n" +
		"Use inline mode to search your collection:"
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

// queuedUpdate carries an incoming update alongside a ready-to-use,
// update-scoped context: the service's long-lived context (so in-flight
// processing isn't cancelled the instant the webhook HTTP handler
// returns, which it does immediately after queuing) as the parent of a
// real span started for that update. Built once at enqueue time, in
// Handler(), where both pieces (the long-lived ctx and the request's
// trace headers) are available -- r.Context() itself is never used past
// that point, since it gets cancelled right after the handler returns.
type queuedUpdate struct {
	ctx    context.Context
	update tgbotapi.Update
}

func (s *TelegramBotServiceImpl) Handler() http.Handler {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	updates := make(chan queuedUpdate, 100)
	go s.dispatchUpdates(ctx, updates)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var update tgbotapi.Update
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		// Span is retrieved back out via trace.SpanFromContext(q.ctx) in
		// dispatchUpdates and ended there, once the dispatched handler
		// actually finishes -- not here.
		updateCtx, _ := tracing.StartHTTP(ctx, "telegram-service", "telegram.update", r.Header)
		updates <- queuedUpdate{ctx: updateCtx, update: update}
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

func (s *TelegramBotServiceImpl) dispatchUpdates(ctx context.Context, updates <-chan queuedUpdate) {
	s.log.InfoContext(ctx, "Authorized", "account", s.bot.Self.UserName)
	for {
		select {
		case <-ctx.Done():
			return
		case q := <-updates:
			update := q.update
			span := trace.SpanFromContext(q.ctx)
			if update.InlineQuery != nil {
				go func() { defer span.End(); s.handleInlineRequest(q.ctx, &update) }()
			} else if update.Message != nil {
				if update.Message.IsCommand() {
					go func() { defer span.End(); s.handleCommand(q.ctx, update.Message) }()
				} else {
					go func() { defer span.End(); s.handleMessage(q.ctx, update.Message) }()
				}
			} else if update.ChosenInlineResult != nil {
				go func() { defer span.End(); s.handleChosenResult(q.ctx, &update) }()
			} else {
				span.End()
			}
		}
	}
}

func (s *TelegramBotServiceImpl) handleCommand(ctx context.Context, requestMessage *tgbotapi.Message) {
	switch requestMessage.Command() {
	case "start":
		var userId int64
		if requestMessage.From != nil {
			userId = requestMessage.From.ID
		}
		var sb strings.Builder
		sb.WriteString(msgStartIntro)
		for _, line := range s.inline.HelpLines(userId) {
			sb.WriteString("\n• @")
			sb.WriteString(s.bot.Self.UserName)
			sb.WriteString(" ")
			sb.WriteString(line)
		}
		msg := tgbotapi.NewMessage(requestMessage.Chat.ID, sb.String())
		msg.ReplyToMessageID = requestMessage.MessageID
		if _, err := s.bot.Send(msg); err != nil {
			s.log.ErrorContext(ctx, "Failed to send start message", "error", err)
		}
	default:
		s.log.InfoContext(ctx, "Unknown command", "command", requestMessage.Command())
	}
}

func (s *TelegramBotServiceImpl) handleMessage(ctx context.Context, requestMessage *tgbotapi.Message) {
	s.log.InfoContext(ctx, "Bot message request")
	s.log.DebugContext(ctx, "Bot message request details",
		"request", requestMessage)

	var processingMsgID int
	processingNotif := tgbotapi.NewMessage(requestMessage.Chat.ID, msgProcessing)
	processingNotif.ReplyToMessageID = requestMessage.MessageID
	if sent, err := s.bot.Send(processingNotif); err == nil {
		processingMsgID = sent.MessageID
	} else {
		s.log.ErrorContext(ctx, "Failed to send processing message", "error", err)
	}

	var answer *MessageHandlerResponse
	var err error
	if requestMessage.Video != nil {
		answer, err = s.message.ProcessVideoMessage(ctx, requestMessage)
	} else if requestMessage.Photo != nil {
		answer, err = s.message.ProcessImageMessage(ctx, requestMessage)
	} else if requestMessage.Document != nil {
		answer, err = s.message.ProcessDocumentMessage(ctx, requestMessage)
	} else if requestMessage.Text != "" && helper.IsYouTubeURL(requestMessage.Text) {
		answer, err = s.message.ProcessYouTubeMessage(ctx, requestMessage)
	} else {
		err = errors.New("message dont contain any media temp")
	}

	if err != nil {
		s.log.ErrorContext(ctx, "Failed to process message", "error", err)
		errText := err.Error()
		if errors.Is(err, ErrForbidden) {
			errText = msgForbidden
		}
		if editErr := s.editMessage(requestMessage.Chat.ID, processingMsgID, errText, ""); editErr != nil {
			errorResponseMessage := tgbotapi.NewMessage(requestMessage.Chat.ID, errText)
			errorResponseMessage.ReplyToMessageID = requestMessage.MessageID
			if _, sendErr := s.bot.Send(errorResponseMessage); sendErr != nil {
				s.log.ErrorContext(ctx, "Failed to send error message", "error", sendErr)
			}
		}
		return
	}

	if err = s.editMessage(requestMessage.Chat.ID, processingMsgID, answer.Message, answer.ParseMode); err != nil {
		s.log.ErrorContext(ctx, "Failed to edit processing message", "error", err)
		if sendErr := s.sendCommonResponseMessage(ctx, requestMessage, answer); sendErr != nil {
			s.log.ErrorContext(ctx, "Failed to send message to bot", "error", sendErr)
			s.sendCommonErrorMessage(ctx, requestMessage, sendErr)
		}
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

func (s *TelegramBotServiceImpl) editMessage(chatID int64, msgID int, text, parseMode string) error {
	if msgID == 0 {
		return errors.New("no message to edit")
	}
	edit := tgbotapi.NewEditMessageText(chatID, msgID, text)
	edit.ParseMode = parseMode
	_, err := s.bot.Send(edit)
	return err
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
	cfg *conf.Config,
) TelegramBotService {
	return &TelegramBotServiceImpl{
		bot:        bot,
		inline:     inline,
		message:    message,
		webhookCfg: cfg.Webhook,
		log:        slog.With("service", "TelegramBotService"),
	}
}
