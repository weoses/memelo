package service

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"mine.local/ocr-gallery/telegram-service/conf"
)

type InlineHandlerService interface {
	ProcessQuery(
		ctx context.Context,
		request *tgbotapi.InlineQuery,
	) (*tgbotapi.InlineConfig, error)
}

type InineHandlerServiceImpl struct {
	userAccount UserAccountService
	storage     StorageConnector
	config      *conf.InlineConfig
}

// ProcessQuery implements InlineService.
func (i *InineHandlerServiceImpl) ProcessQuery(
	ctx context.Context,
	request *tgbotapi.InlineQuery,
) (*tgbotapi.InlineConfig, error) {
	userId := request.From.ID
	query := request.Query

	accountId, err := i.userAccount.MapUserToAccount(ctx, userId)
	if err != nil {
		return nil, err
	}

	var searchAfter *uuid.UUID
	if request.Offset != "" {
		offset, err := uuid.Parse(request.Offset)
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
		return nil, err
	}

	if results == nil {
		retval := tgbotapi.InlineConfig{
			InlineQueryID: request.ID,
			CacheTime:     5,
			IsPersonal:    true,
		}
		return &retval, nil
	}

	resultsUnr := *results

	photos := make([]interface{}, len(resultsUnr))
	for index, item := range resultsUnr {
		inlineChoice := tgbotapi.NewInlineQueryResultPhotoWithThumb(
			item.Id.String(),
			*item.ImageUrl,
			*item.ImageUrl,
		)
		inlineChoice.MimeType = "image/jpeg"
		inlineChoice.Height = *item.Thumbnail.ThumbHeight
		inlineChoice.Width = *item.Thumbnail.ThumbWidth

		photos[index] = inlineChoice
	}

	nextOffset := ""
	if len(resultsUnr) == i.config.PageSize {
		nextOffset = resultsUnr[i.config.PageSize].Id.String()
	}

	retval := tgbotapi.InlineConfig{
		InlineQueryID: request.ID,
		CacheTime:     5,
		IsPersonal:    true,
		NextOffset:    nextOffset,
	}
	retval.Results = photos

	return &retval, nil
}

func substr(s string, start, end int) string {
	counter, startIdx := 0, 0
	for i := range s {
		if counter == start {
			startIdx = i
		}
		if counter == end {
			return s[startIdx:i]
		}
		counter++
	}
	return s[startIdx:]
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
