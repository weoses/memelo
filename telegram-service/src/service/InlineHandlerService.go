package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/weoses/memelo/common/commonconst"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/weoses/memelo/telegram-service/conf"
)

const inlineDeletePrefix = "!del"

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
	config      *conf.InlineConfig
}

func (i *InineHandlerServiceImpl) ProcessChosenInlineQuery(ctx context.Context, request *tgbotapi.ChosenInlineResult) error {
	if !strings.HasPrefix(request.Query, inlineDeletePrefix) {
		return nil
	}

	imageId := request.ResultID
	userId := request.From.ID
	slog.Info("Inline result : del query:",
		"userId", userId,
		commonconst.QUERY_LOG, request.Query,
		"messageId", request.InlineMessageID)

	accountId, err := i.userAccount.MapUserToAccount(ctx, userId)
	if err != nil {
		return fmt.Errorf("ProcessChosenInlineQuery: MapUserToAccount failed, userId=%d, err=%w", userId, err)
	}

	imageIdUuid, err := uuid.Parse(imageId)

	err = i.storage.DeleteMeme(ctx, accountId, imageIdUuid)
	if err != nil {
		return fmt.Errorf("ProcessChosenInlineQuery: DeleteMeme failed, accountId=%s, memeId=%s err=%w",
			accountId, imageIdUuid, err)
	}

	return nil
}

// ProcessQuery implements InlineService.
func (i *InineHandlerServiceImpl) ProcessQuery(
	ctx context.Context,
	request *tgbotapi.InlineQuery,
) (*tgbotapi.InlineConfig, error) {
	userId := request.From.ID
	query := request.Query
	delQuery := false

	slog.Info("Inline query:",
		"userId", userId,
		"requestId", request.ID,
		"query", request.Query,
		"offset", request.Offset)

	if strings.HasPrefix(query, inlineDeletePrefix) {
		query = strings.TrimPrefix(query, inlineDeletePrefix)
		delQuery = true
	}

	query = strings.TrimSpace(query)

	accountId, err := i.userAccount.MapUserToAccount(ctx, userId)
	if err != nil {
		return nil, err
	}

	var searchAfter *int64
	if request.Offset != "" {
		offset, err := strconv.ParseInt(request.Offset, 10, 64)
		if err != nil {
			return nil, err
		}
		searchAfter = &offset
	}

	results, err := i.storage.ProcessSearchQuery(
		ctx,
		accountId,
		query,
		i.config.PageSize,
		searchAfter,
	)
	if err != nil {
		return nil, fmt.Errorf("ProcessSearchQuery failed: %w", err)
	}

	slog.Info("Search query result",
		"userId", userId,
		"requestId", request.ID,
		"resultListSize", len(results))

	if results == nil {
		retval := tgbotapi.InlineConfig{
			InlineQueryID: request.ID,
			CacheTime:     120,
			IsPersonal:    true,
		}
		return &retval, nil
	}

	photos := make([]interface{}, len(results))
	for index, item := range results {
		slog.Debug("SearchResultItem ",
			"userId", userId,
			"requestId", request.ID,
			"index", index,
			"id", item.Id,
			"sortId", item.SortId,
			"url", item.ImageUrl,
		)

		inlineChoice := tgbotapi.NewInlineQueryResultPhotoWithThumb(
			item.Id.String(),
			item.ImageUrl,
			item.ImageUrl,
		)
		inlineChoice.MimeType = "image/jpeg"
		inlineChoice.Height = item.ThumbHeight
		inlineChoice.Width = item.ThumbWidth

		if delQuery {
			inlineChoice.Caption = "Deleted"
		}

		photos[index] = inlineChoice
	}

	nextOffset := ""
	if len(results) == i.config.PageSize && i.config.PageSize > 0 {
		nextOffset = strconv.FormatInt(results[i.config.PageSize-1].SortId, 10)
	}

	slog.Info("Search next offset ",
		"userId", userId,
		"requestId", request.ID,
		"nextOffset", nextOffset)

	retval := tgbotapi.InlineConfig{
		InlineQueryID: request.ID,
		CacheTime:     5,
		IsPersonal:    true,
		NextOffset:    nextOffset,
	}
	retval.Results = photos

	return &retval, nil
}

func NewInlineService(
	userAccount UserAccountService,
	storage StorageConnector,
	config *conf.InlineConfig,
) InlineHandlerService {

	return &InineHandlerServiceImpl{
		userAccount: userAccount,
		storage:     storage,
		config:      config,
	}
}
