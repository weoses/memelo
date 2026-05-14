package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/weoses/memelo/telegram-service/util"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/weoses/memelo/telegram-service/conf"
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
}

type InineHandlerServiceImpl struct {
	userAccount UserAccountService
	storage     StorageConnector
	config      *conf.Config
	log         *slog.Logger
	factory     QueryProcessorFactory
}

func (i *InineHandlerServiceImpl) ProcessChosenInlineQuery(ctx context.Context, request *tgbotapi.ChosenInlineResult) error {
	userId := request.From.ID
	i.log.InfoContext(ctx, "Inline result chosen",
		"userId", userId,
		"query", request.Query,
		"messageId", request.InlineMessageID)

	processor := i.factory.GetProcessor(request.Query)

	accountId, err := i.userAccount.MapUserToAccount(ctx, userId)
	if err != nil {
		return fmt.Errorf("ProcessChosenInlineQuery: MapUserToAccount failed, userId=%d: %w", userId, err)
	}

	return processor.ProcessChosenQuery(ctx, accountId, request.ResultID)
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

	accountId, err := i.userAccount.MapUserToAccount(ctx, userId)
	if err != nil {
		return nil, err
	}

	processor := i.factory.GetProcessor(request.Query)

	result, err := processor.Process(ctx, accountId, request.Query, util.ParseOffset(request.Offset))
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
}

func NewInlineService(
	userAccount UserAccountService,
	storage StorageConnector,
	config *conf.Config,
	factory QueryProcessorFactory,
) InlineHandlerService {

	return &InineHandlerServiceImpl{
		userAccount: userAccount,
		storage:     storage,
		config:      config,
		log:         slog.With("service", "InlineHandlerService"),
		factory:     factory,
	}
}
