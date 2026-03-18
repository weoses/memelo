package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
	log                *slog.Logger
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

// ProcessImageMessage implements MessageHandlerService.
func (m MessageHandlerServiceImpl) ProcessImageMessage(ctx context.Context, message *tgbotapi.Message) (*MessageHandlerResponse, error) {
	var fileId string
	if len(message.Photo) >= 1 {
		fileId = message.Photo[len(message.Photo)-1].FileID
	}

	if fileId == "" {
		return nil, errors.New("messageHandlerService: message dont contain image")
	}

	file, err := m.fileResolver.GetFile(ctx, fileId)
	if err != nil {
		return nil, fmt.Errorf("messageHandlerService: GetFile failed, fileId: %s : %w", fileId, err)
	}

	accountId, err := m.userAccountService.MapUserToAccount(ctx, message.Chat.ID)
	if err != nil {
		return nil, fmt.Errorf("messageHandlerService: MapUserToAccount failed : %w", err)
	}

	result, err := m.storage.CreateMeme(ctx, file, "image/jpeg", accountId)
	if err != nil {
		return nil, fmt.Errorf("messageHandlerService: CreateMeme failed : %w", err)
	}

	m.log.InfoContext(ctx, "meme created",
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

	file, err := m.fileResolver.GetFile(ctx, message.Video.FileID)
	if err != nil {
		return nil, fmt.Errorf("messageHandlerService: GetFile failed, fileId: %s : %w", message.Video.FileID, err)
	}

	accountId, err := m.userAccountService.MapUserToAccount(ctx, message.Chat.ID)
	if err != nil {
		return nil, fmt.Errorf("messageHandlerService: MapUserToAccount failed : %w", err)
	}

	result, err := m.storage.CreateVideo(ctx, file, accountId)
	if err != nil {
		return nil, fmt.Errorf("messageHandlerService: CreateVideo failed : %w", err)
	}

	m.log.InfoContext(ctx, "video meme created",
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

func NewMessageHandlerService(
	storage StorageConnector,
	fileResolver TelegramFileResolverService,
	userAccountService UserAccountService,
) MessageHandlerService {
	return &MessageHandlerServiceImpl{
		storage:            storage,
		fileResolver:       fileResolver,
		userAccountService: userAccountService,
		log:                slog.With("service", "MessageHandlerService"),
	}
}
