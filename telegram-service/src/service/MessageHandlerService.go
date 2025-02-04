package service

import (
	"context"
	"errors"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type MessageHandlerService interface {
	ProcessMessage(message *tgbotapi.Message) (*MessageHandlerResponse, error)
}

type MessageHandlerResponse struct {
	Message   string
	ParseMode string
}

type MessageHandlerServiceImpl struct {
	storage            StorageConnector
	fileResolver       TelegramFileResolverService
	userAccountService UserAccountService
}

// ProcessMessage implements MessageHandlerService.
func (m MessageHandlerServiceImpl) ProcessMessage(message *tgbotapi.Message) (*MessageHandlerResponse, error) {
	var fileId string
	if len(message.Photo) >= 1 {
		fileId = message.Photo[len(message.Photo)-1].FileID
	}

	if fileId == "" {
		return nil, errors.New("message dont contain image")
	}

	file, err := m.fileResolver.GetFile(fileId)

	if err != nil {
		return nil, err
	}

	accountId, err := m.userAccountService.MapUserToAccount(context.TODO(), message.Chat.ID)
	if err != nil {
		return nil, err
	}

	result, err := m.storage.CreateMeme(file, "image/jpeg", accountId)
	if err != nil {
		return nil, err
	}

	return &MessageHandlerResponse{
		Message:   fmt.Sprintf("\n```Text\n%s\n```\n ID %s", result.Text, result.Id),
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
	}
}
