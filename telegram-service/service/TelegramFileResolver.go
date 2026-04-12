package service

import (
	"context"
	"fmt"
	"log/slog"
)

type BotFileGetter interface {
	GetFileDirectURL(fileID string) (string, error)
}

type TelegramFileResolverService interface {
	GetFileURL(ctx context.Context, fileID string) (string, error)
}

type TelegramFileResolverServiceImpl struct {
	fileGetter BotFileGetter
	log        *slog.Logger
}

func (t *TelegramFileResolverServiceImpl) GetFileURL(ctx context.Context, fileID string) (string, error) {
	url, err := t.fileGetter.GetFileDirectURL(fileID)
	if err != nil {
		return "", fmt.Errorf("TelegramFileResolverService: GetFileDirectURL failed, fileId: %s : %w", fileID, err)
	}
	return url, nil
}

func NewTelegramFileResolverService(fileGetter BotFileGetter) TelegramFileResolverService {
	return &TelegramFileResolverServiceImpl{
		fileGetter: fileGetter,
		log:        slog.With("service", "TelegramFileResolverService"),
	}
}
