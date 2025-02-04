package service

import (
	"context"
	"log"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

	log.Printf("Inline query: userid: '%d' id: '%s' text: '%s' offset: '%s'",
		userId, request.ID, request.Query, request.Offset)

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

	photos := make([]interface{}, len(results))
	for index, item := range results {
		inlineChoice := tgbotapi.NewInlineQueryResultPhotoWithThumb(
			item.Id.String(),
			item.ImageUrl,
			item.ImageUrl,
		)
		inlineChoice.MimeType = "image/jpeg"
		inlineChoice.Height = item.ThumbHeight
		inlineChoice.Width = item.ThumbWidth

		photos[index] = inlineChoice
	}

	log.Printf("Result count is %d", len(results))

	nextOffset := ""
	if len(results) == i.config.PageSize && i.config.PageSize > 0 {
		nextOffset = strconv.FormatInt(results[i.config.PageSize-1].SortId, 10)
		log.Printf("Next offset is %s", nextOffset)
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
