package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/telegram-service/conf"
	"github.com/weoses/memelo/telegram-service/entity"
)

type QueryProcessorResult struct {
	Results    []interface{}
	Pagination *entity.PaginationOffset
	CacheTime  int
}

type QueryProcessor interface {
	Process(ctx context.Context, accountId uuid.UUID, query string, pagination *entity.PaginationOffset) (*QueryProcessorResult, error)
	ProcessChosenQuery(ctx context.Context, accountId uuid.UUID, resultId string) error
}

type QueryProcessorFactory interface {
	GetProcessor(rawQuery string) QueryProcessor
}

func buildTelegramResults(ctx context.Context, log *slog.Logger, results []entity.MemeSearchResult, markDeleted bool) ([]interface{}, error) {
	return helper.TransformSliceErr(
		results,
		make([]interface{}, len(results)),
		func(item entity.MemeSearchResult) (interface{}, error) {
			log.DebugContext(ctx, "SearchResultItem", "id", item.Id, "url", item.MediaUrl)

			caption := item.Caption
			if caption == "" {
				caption = "media"
			}

			switch item.Type {
			case entity.ResultTypeImage:
				inlineChoice := tgbotapi.NewInlineQueryResultPhotoWithThumb(item.Id, item.MediaUrl, item.ThumbUrl)
				inlineChoice.MimeType = "image/jpeg"
				inlineChoice.Height = item.ThumbHeight
				inlineChoice.Width = item.ThumbWidth
				if markDeleted {
					inlineChoice.Caption = "Deleted"
				}
				return inlineChoice, nil

			case entity.ResultTypeVideo:
				inlineChoice := tgbotapi.NewInlineQueryResultVideo(item.Id, item.MediaUrl)
				inlineChoice.MimeType = "video/mp4"
				inlineChoice.ThumbURL = item.ThumbUrl
				inlineChoice.Width = item.ThumbWidth
				inlineChoice.Height = item.ThumbHeight
				inlineChoice.Title = caption
				if markDeleted {
					inlineChoice.Caption = "Deleted"
				}
				return inlineChoice, nil
			}
			return nil, fmt.Errorf("buildTelegramResults: unknown result type %q", item.Type)
		},
	)
}

// SearchQueryProcessor handles plain search queries.
type SearchQueryProcessor struct {
	storage StorageConnector
	config  *conf.Config
	log     *slog.Logger
}

func (p *SearchQueryProcessor) Process(ctx context.Context, accountId uuid.UUID, query string, pagination *entity.PaginationOffset) (*QueryProcessorResult, error) {
	params := entity.SearchParams{Query: strings.TrimSpace(query), Pagination: pagination}
	searchResult, err := p.storage.ProcessSearchQuery(ctx, accountId, params, p.config.Inline.PageSize)
	if err != nil {
		return nil, fmt.Errorf("SearchQueryProcessor: %w", err)
	}

	results, err := buildTelegramResults(ctx, p.log, searchResult.Results, false)
	if err != nil {
		return nil, err
	}

	var nextPagination *entity.PaginationOffset
	if len(results) == p.config.Inline.PageSize && p.config.Inline.PageSize > 0 {
		nextPagination = searchResult.Pagination
	}
	return &QueryProcessorResult{Results: results, Pagination: nextPagination, CacheTime: 120}, nil
}

func (p *SearchQueryProcessor) ProcessChosenQuery(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

// DeleteQueryProcessor handles /delete queries: shows results marked for deletion,
// and performs the actual delete when a result is chosen.
type DeleteQueryProcessor struct {
	storage StorageConnector
	config  *conf.Config
	log     *slog.Logger
}

func (p *DeleteQueryProcessor) Process(ctx context.Context, accountId uuid.UUID, query string, pagination *entity.PaginationOffset) (*QueryProcessorResult, error) {
	params := entity.SearchParams{Query: strings.TrimSpace(strings.TrimPrefix(query, inlineDeletePrefix)), Pagination: pagination}
	searchResult, err := p.storage.ProcessSearchQuery(ctx, accountId, params, p.config.Inline.PageSize)
	if err != nil {
		return nil, fmt.Errorf("DeleteQueryProcessor: %w", err)
	}

	results, err := buildTelegramResults(ctx, p.log, searchResult.Results, true)
	if err != nil {
		return nil, err
	}

	var nextPagination *entity.PaginationOffset
	if len(results) == p.config.Inline.PageSize && p.config.Inline.PageSize > 0 {
		nextPagination = searchResult.Pagination
	}
	return &QueryProcessorResult{Results: results, Pagination: nextPagination, CacheTime: 120}, nil
}

func (p *DeleteQueryProcessor) ProcessChosenQuery(ctx context.Context, accountId uuid.UUID, resultId string) error {
	memeIdUuid, err := uuid.Parse(resultId)
	if err != nil {
		return fmt.Errorf("DeleteQueryProcessor: failed to parse meme uuid %q: %w", resultId, err)
	}
	err = p.storage.DeleteMeme(ctx, accountId, memeIdUuid)
	if err != nil {
		return fmt.Errorf("DeleteQueryProcessor: DeleteMeme failed, accountId=%s, memeId=%s: %w", accountId, memeIdUuid, err)
	}
	return nil
}

// RandomQueryProcessor handles /random [image|video] queries.
// The query parameter is a media type specifier ("image", "video", or ""), not a search term.
// Pagination is intentionally ignored — random results have no meaningful cursor.
type RandomQueryProcessor struct {
	storage StorageConnector
	log     *slog.Logger
}

func (p *RandomQueryProcessor) Process(ctx context.Context, accountId uuid.UUID, query string, _ *entity.PaginationOffset) (*QueryProcessorResult, error) {
	mediaType := strings.TrimSpace(strings.TrimPrefix(query, randomPrefix))
	if mediaType != entity.ResultTypeImage && mediaType != entity.ResultTypeVideo {
		mediaType = ""
	}

	meme, err := p.storage.GetRandomMeme(ctx, accountId, mediaType)
	if err != nil {
		return nil, fmt.Errorf("RandomQueryProcessor: %w", err)
	}
	if meme == nil {
		return &QueryProcessorResult{}, nil
	}

	results, err := buildTelegramResults(ctx, p.log, []entity.MemeSearchResult{*meme}, false)
	if err != nil {
		return nil, err
	}
	return &QueryProcessorResult{Results: results, CacheTime: 1}, nil
}

func (p *RandomQueryProcessor) ProcessChosenQuery(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

type QueryProcessorFactoryImpl struct {
	searchProcessor QueryProcessor
	deleteProcessor QueryProcessor
	randomProcessor QueryProcessor
}

func (f *QueryProcessorFactoryImpl) GetProcessor(rawQuery string) QueryProcessor {
	switch {
	case strings.HasPrefix(rawQuery, inlineDeletePrefix):
		return f.deleteProcessor
	case strings.HasPrefix(rawQuery, randomPrefix):
		return f.randomProcessor
	default:
		return f.searchProcessor
	}
}

func NewQueryProcessorFactory(storage StorageConnector, config *conf.Config) QueryProcessorFactory {
	return &QueryProcessorFactoryImpl{
		searchProcessor: &SearchQueryProcessor{
			storage: storage,
			config:  config,
			log:     slog.With("service", "SearchQueryProcessor"),
		},
		deleteProcessor: &DeleteQueryProcessor{
			storage: storage,
			config:  config,
			log:     slog.With("service", "DeleteQueryProcessor"),
		},
		randomProcessor: &RandomQueryProcessor{
			storage: storage,
			log:     slog.With("service", "RandomQueryProcessor"),
		},
	}
}
