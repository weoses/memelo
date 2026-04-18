package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/telegram-service/entity"

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
	config      *conf.Config
	log         *slog.Logger
}

func (i *InineHandlerServiceImpl) ProcessChosenInlineQuery(ctx context.Context, request *tgbotapi.ChosenInlineResult) error {
	if !strings.HasPrefix(request.Query, inlineDeletePrefix) {
		return nil
	}
	var accountId uuid.UUID
	var imageIdUuid uuid.UUID
	var err error

	imageId := request.ResultID
	userId := request.From.ID
	i.log.InfoContext(ctx, "Inline result : del query:",
		"userId", userId,
		"query", request.Query,
		"messageId", request.InlineMessageID)

	accountId, err = i.userAccount.MapUserToAccount(ctx, userId)
	if err != nil {
		return fmt.Errorf("ProcessChosenInlineQuery: MapUserToAccount failed, userId=%d, err=%w", userId, err)
	}

	imageIdUuid, err = uuid.Parse(imageId)
	if err != nil {
		return fmt.Errorf("failed to parse uuid %w", err)
	}
	err = i.storage.DeleteMeme(ctx, accountId, imageIdUuid)
	if err != nil {
		return fmt.Errorf("ProcessChosenInlineQuery: DeleteMeme failed, accountId=%s, memeId=%s err=%w",
			accountId, imageIdUuid, err)
	}

	return nil
}

func writeStr(w io.Writer, s string) error {
	if err := binary.Write(w, binary.LittleEndian, uint32(len(s))); err != nil {
		return err
	}
	_, err := io.WriteString(w, s)
	return err
}

func readStr(r io.Reader) (string, error) {
	var n uint32
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return "", err
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", err
	}
	return string(b), nil
}

func parseOffset(offset string) *entity.PaginationOffset {
	if offset == "" {
		return nil
	}
	raw, err := base64.StdEncoding.DecodeString(offset)
	if err != nil {
		return nil
	}
	r := bytes.NewReader(raw)
	searcher, err := readStr(r)
	if err != nil {
		return nil
	}
	var mapLen uint32
	if err := binary.Read(r, binary.LittleEndian, &mapLen); err != nil {
		return nil
	}
	sortingAfter := make(map[string]string, mapLen)
	for range mapLen {
		k, err := readStr(r)
		if err != nil {
			return nil
		}
		v, err := readStr(r)
		if err != nil {
			return nil
		}
		sortingAfter[k] = v
	}
	return &entity.PaginationOffset{Searcher: searcher, SortingAfter: sortingAfter}
}

func serializeOffset(p *entity.PaginationOffset) string {
	if p == nil || (p.Searcher == "" && len(p.SortingAfter) == 0) {
		return ""
	}
	var buf bytes.Buffer
	if writeStr(&buf, p.Searcher) != nil {
		return ""
	}
	if binary.Write(&buf, binary.LittleEndian, uint32(len(p.SortingAfter))) != nil {
		return ""
	}
	for k, v := range p.SortingAfter {
		if writeStr(&buf, k) != nil || writeStr(&buf, v) != nil {
			return ""
		}
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// ProcessQuery implements InlineService.
func (i *InineHandlerServiceImpl) ProcessQuery(
	ctx context.Context,
	request *tgbotapi.InlineQuery,
) (*tgbotapi.InlineConfig, error) {
	userId := request.From.ID
	query := request.Query
	delQuery := false

	i.log.InfoContext(ctx, "ProcessQuery start:",
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

	params := entity.SearchParams{
		Query:      query,
		Pagination: parseOffset(request.Offset),
	}

	searchResult, err := i.storage.ProcessSearchQuery(
		ctx,
		accountId,
		params,
		i.config.Inline.PageSize,
	)
	if err != nil {
		return nil, fmt.Errorf("ProcessSearchQuery failed: %w", err)
	}

	results := searchResult.Results

	i.log.InfoContext(ctx, "Search query result",
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

	photos := helper.TransformSlice(
		results,
		make([]interface{}, len(results)),
		func(item entity.MemeSearchResult) interface{} {
			i.log.DebugContext(ctx, "SearchResultItem",
				"id", item.Id,
				"url", item.MediaUrl,
			)

			caption := item.Caption
			if caption == "" {
				caption = "media"
			}

			switch item.Type {
			case entity.ResultTypeImage:
				{
					inlineChoice := tgbotapi.NewInlineQueryResultPhotoWithThumb(
						item.Id,
						item.MediaUrl,
						item.ThumbUrl,
					)
					inlineChoice.MimeType = "image/jpeg"
					inlineChoice.Height = item.ThumbHeight
					inlineChoice.Width = item.ThumbWidth
					if delQuery {
						inlineChoice.Caption = "Deleted"
					}
					return inlineChoice
				}

			case entity.ResultTypeVideo:
				{
					inlineChoice := tgbotapi.NewInlineQueryResultVideo(
						item.Id,
						item.MediaUrl)
					inlineChoice.MimeType = "video/mp4"
					inlineChoice.ThumbURL = item.ThumbUrl
					inlineChoice.Width = item.ThumbWidth
					inlineChoice.Height = item.ThumbHeight
					inlineChoice.Title = caption

					if delQuery {
						inlineChoice.Caption = "Deleted"
					}
					return inlineChoice
				}
			}
			panic("unknown result type")
		})

	nextOffset := ""
	if len(results) == i.config.Inline.PageSize && i.config.Inline.PageSize > 0 {
		nextOffset = serializeOffset(searchResult.Pagination)
	}

	i.log.InfoContext(ctx, "Search next offset",
		"userId", userId,
		"requestId", request.ID,
		"nextOffset", nextOffset)

	retval := tgbotapi.InlineConfig{
		InlineQueryID: request.ID,
		CacheTime:     50,
		IsPersonal:    true,
		NextOffset:    nextOffset,
	}
	retval.Results = photos

	return &retval, nil
}

func NewInlineService(
	userAccount UserAccountService,
	storage StorageConnector,
	config *conf.Config,
) InlineHandlerService {

	return &InineHandlerServiceImpl{
		userAccount: userAccount,
		storage:     storage,
		config:      config,
		log:         slog.With("service", "InlineHandlerService"),
	}
}
