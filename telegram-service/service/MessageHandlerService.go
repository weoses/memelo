package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/weoses/memelo/common/helper"
	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/common/temp"
)

type MessageHandlerService interface {
	ProcessImageMessage(ctx context.Context, message *tgbotapi.Message) (*MessageHandlerResponse, error)
	ProcessVideoMessage(ctx context.Context, message *tgbotapi.Message) (*MessageHandlerResponse, error)
	ProcessCommandAddTag(ctx context.Context, message *tgbotapi.Message) (*MessageHandlerResponse, error)
}

type MessageHandlerResponse struct {
	Message   string
	ParseMode string
}

type MessageHandlerServiceImpl struct {
	storage            StorageConnector
	fileResolver       TelegramFileResolverService
	userAccountService UserAccountService
	tmpDataService     commonservice.TmpDataService
	slogger            *slog.Logger
}

func (m MessageHandlerServiceImpl) ProcessCommandAddTag(ctx context.Context, message *tgbotapi.Message) (*MessageHandlerResponse, error) {
	arguments := message.CommandArguments()
	if arguments == "" {
		return nil, errors.New("empty arguments for add tag, need NAME DESCRIPTION")
	}
	args := strings.SplitN(arguments, " ", 2)
	if len(args) < 2 {
		return nil, errors.New("invalid arguments for add tag, need NAME DESCRIPTION")
	}

	name := args[0]
	description := args[1]
	if len(name) == 0 || len(description) == 0 {
		return nil, errors.New("empty arguments for add tag, need NAME DESCRIPTION")
	}

	accountId, err := m.userAccountService.MapUserToAccount(ctx, message.Chat.ID)
	if err != nil {
		return nil, fmt.Errorf("messageHandlerService: MapUserToAccount failed: %w", err)
	}

	if err := m.storage.AddTag(ctx, accountId, name, description); err != nil {
		return nil, fmt.Errorf("messageHandlerService: AddTag failed: %w", err)
	}

	return &MessageHandlerResponse{
		Message:   fmt.Sprintf("Tag `%s` created", name),
		ParseMode: "Markdown",
	}, nil
}

func (m MessageHandlerServiceImpl) ProcessImageMessage(ctx context.Context, message *tgbotapi.Message) (*MessageHandlerResponse, error) {
	var fileId string
	if len(message.Photo) >= 1 {
		fileId = message.Photo[len(message.Photo)-1].FileID
	}

	if fileId == "" {
		return nil, errors.New("messageHandlerService: message dont contain image")
	}

	fileURL, err := m.fileResolver.GetFileURL(ctx, fileId)
	if err != nil {
		return nil, fmt.Errorf("messageHandlerService: GetFileURL failed, fileId: %s : %w", fileId, err)
	}

	s3data, err := m.downloadToS3(ctx, fileURL)
	if err != nil {
		return nil, fmt.Errorf("messageHandlerService: downloadToS3 failed: %w", err)
	}
	defer helper.QuietClose(s3data, m.slogger)

	accountId, err := m.userAccountService.MapUserToAccount(ctx, message.Chat.ID)
	if err != nil {
		return nil, fmt.Errorf("messageHandlerService: MapUserToAccount failed : %w", err)
	}

	result, err := m.storage.CreateMeme(ctx, s3data, "image/jpeg", accountId)
	if err != nil {
		return nil, fmt.Errorf("messageHandlerService: CreateMeme failed : %w", err)
	}

	m.slogger.InfoContext(ctx, "meme created",
		"imageId", result.Id,
		"duplicate", result.DuplicateStatus)

	return &MessageHandlerResponse{
		Message: fmt.Sprintf(
			"\n```Text\n%s\n```\n ID: `%s` \n Status: `%s`\n Tags: ```%s```",
			result.Text,
			result.Id,
			result.DuplicateStatus,
			strings.Join(result.Tags, ", ")),
		ParseMode: "Markdown",
	}, nil
}

func (m MessageHandlerServiceImpl) ProcessVideoMessage(ctx context.Context, message *tgbotapi.Message) (*MessageHandlerResponse, error) {
	if message.Video == nil {
		return nil, errors.New("messageHandlerService: message does not contain a video")
	}

	fileURL, err := m.fileResolver.GetFileURL(ctx, message.Video.FileID)
	if err != nil {
		return nil, fmt.Errorf("messageHandlerService: GetFileURL failed, fileId: %s : %w", message.Video.FileID, err)
	}

	s3data, err := m.downloadToS3(ctx, fileURL)
	if err != nil {
		return nil, fmt.Errorf("messageHandlerService: downloadToS3 failed: %w", err)
	}
	defer helper.QuietClose(s3data, m.slogger)

	accountId, err := m.userAccountService.MapUserToAccount(ctx, message.Chat.ID)
	if err != nil {
		return nil, fmt.Errorf("messageHandlerService: MapUserToAccount failed : %w", err)
	}

	result, err := m.storage.CreateVideo(ctx, s3data, accountId)
	if err != nil {
		return nil, fmt.Errorf("messageHandlerService: CreateVideo failed : %w", err)
	}

	m.slogger.InfoContext(ctx, "video meme created",
		"memeId", result.Id,
		"duplicate", result.DuplicateStatus)

	return &MessageHandlerResponse{
		Message: fmt.Sprintf(
			"\n```Text\n%s\n```\n ID: `%s` \n Status: `%s`\n Tags: ```%s```",
			result.Text,
			result.Id,
			result.DuplicateStatus,
			strings.Join(result.Tags, ", ")),
		ParseMode: "Markdown",
	}, nil
}

func (m MessageHandlerServiceImpl) downloadToS3(ctx context.Context, fileURL string) (temp.S3BackedData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("downloadToS3: create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloadToS3: http get: %w", err)
	}
	defer helper.QuietClose(resp.Body, m.slogger)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("downloadToS3: non-2xx status: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	contentType = strings.SplitAfter(contentType, ";")[0]

	s3data, err := m.tmpDataService.ByReader(ctx, contentType, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("downloadToS3: upload to s3: %w", err)
	}

	return s3data, nil
}

func NewMessageHandlerService(
	storage StorageConnector,
	fileResolver TelegramFileResolverService,
	userAccountService UserAccountService,
	tmpDataService commonservice.TmpDataService,
) MessageHandlerService {
	return &MessageHandlerServiceImpl{
		storage:            storage,
		fileResolver:       fileResolver,
		userAccountService: userAccountService,
		tmpDataService:     tmpDataService,
		slogger:            slog.With("service", "MessageHandlerService"),
	}
}
