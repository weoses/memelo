package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/weoses/memelo/telegram-service/conf"
	"github.com/weoses/memelo/telegram-service/util"
)

const inlineDeletePrefix = "/delete"
const inlineRecomputePrefix = "/recompute"
const randomPrefix = "/random"

type InlineHandlerService interface {
	ProcessQuery(
		ctx context.Context,
		request *tgbotapi.InlineQuery,
	) (*tgbotapi.InlineConfig, error)

	ProcessChosenInlineQuery(
		ctx context.Context,
		request *tgbotapi.ChosenInlineResult,
	) error

	HelpLines(userId int64) []string
}

type InineHandlerServiceImpl struct {
	permissionService PermissionService
	storage           StorageConnector
	config            *conf.Config
	log               *slog.Logger
	factory           QueryProcessorFactory
	staticAccountId   uuid.UUID
}

func permissionForQuery(query string) string {
	switch {
	case strings.HasPrefix(query, inlineDeletePrefix):
		return PermissionDelete
	case strings.HasPrefix(query, inlineRecomputePrefix):
		return PermissionRecompute
	default:
		return PermissionSearch
	}
}

func (i *InineHandlerServiceImpl) ProcessChosenInlineQuery(ctx context.Context, request *tgbotapi.ChosenInlineResult) error {
	userId := request.From.ID
	i.log.InfoContext(ctx, "Inline result chosen",
		"userId", userId,
		"query", request.Query,
		"messageId", request.InlineMessageID)

	processor := i.factory.GetProcessor(request.Query)
	permission := permissionForQuery(request.Query)

	_, err := InvokeWithPermission(ctx, i.permissionService, userId, permission, func() (struct{}, error) {
		return struct{}{}, processor.ProcessChosenQuery(ctx, i.staticAccountId, request.ResultID)
	})
	return err
}

func (i *InineHandlerServiceImpl) HelpLines(userId int64) []string {
	var lines []string
	for _, entry := range i.factory.Entries() {
		if i.permissionService.IsAllowed(userId, entry.Permission) {
			lines = append(lines, entry.Processor.HelpText())
		}
	}
	return lines
}

// ProcessQuery implements InlineService.
func (i *InineHandlerServiceImpl) ProcessQuery(
	ctx context.Context,
	request *tgbotapi.InlineQuery,
) (*tgbotapi.InlineConfig, error) {
	userId := request.From.ID

	i.log.InfoContext(ctx, "ProcessQuery start:",
		"query", request.Query,
		"offset", request.Offset)

	processor := i.factory.GetProcessor(request.Query)
	permission := permissionForQuery(request.Query)

	cfg, err := InvokeWithPermission(ctx, i.permissionService, userId, permission, func() (*tgbotapi.InlineConfig, error) {
		result, err := processor.Process(ctx, i.staticAccountId, request.Query, util.ParseOffset(request.Offset))
		if err != nil {
			return nil, fmt.Errorf("ProcessQuery: processor failed: %w", err)
		}

		i.log.InfoContext(ctx, "Process query result",
			"userId", userId,
			"requestId", request.ID,
			"resultListSize", len(result.Results))

		nextOffset := ""
		if result.Pagination != nil {
			nextOffset = util.SerializeOffset(result.Pagination)
		}

		i.log.InfoContext(ctx, "Search next offset",
			"userId", userId,
			"requestId", request.ID,
			"nextOffset", nextOffset)

		return &tgbotapi.InlineConfig{
			InlineQueryID: request.ID,
			CacheTime:     result.CacheTime,
			IsPersonal:    true,
			NextOffset:    nextOffset,
			Results:       result.Results,
		}, nil
	})
	if errors.Is(err, ErrForbidden) {
		return &tgbotapi.InlineConfig{
			InlineQueryID: request.ID,
			Results: []interface{}{
				tgbotapi.InlineQueryResultArticle{
					Type:  "article",
					ID:    "forbidden",
					Title: msgForbidden,
					InputMessageContent: tgbotapi.InputTextMessageContent{
						Text: msgForbidden,
					},
				},
			},
		}, nil
	}
	return cfg, err
}

func NewInlineService(
	permissionService PermissionService,
	storage StorageConnector,
	config *conf.Config,
	factory QueryProcessorFactory,
) InlineHandlerService {
	return &InineHandlerServiceImpl{
		permissionService: permissionService,
		storage:           storage,
		config:            config,
		log:               slog.With("service", "InlineHandlerService"),
		factory:           factory,
		staticAccountId:   uuid.MustParse(config.UserAccount.StaticUuid),
	}
}
