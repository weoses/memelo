package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

type BotFileGetter interface {
	GetFileDirectURL(fileID string) (string, error)
}

type TelegramFileResolverService interface {
	GetFile(ctx context.Context, fileID string) ([]byte, error)
}

type TelegramFileResolverServiceImpl struct {
	fileGetter BotFileGetter
	log        *slog.Logger
}

// GetFile implements TelegramFileResolverService.
func (t *TelegramFileResolverServiceImpl) GetFile(ctx context.Context, fileID string) ([]byte, error) {
	url, err := t.fileGetter.GetFileDirectURL(fileID)
	if err != nil {
		return nil, fmt.Errorf("TelegramFileResolverService: GetFileDirectURL failed, fileId: %s : %w", fileID, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("TelegramFileResolverService: create request failed, url: %s : %w", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("TelegramFileResolverService: download file by got url failed, url: %s : %w", url, err)
	}

	if resp.StatusCode >= 400 {
		return nil, errors.New("TelegramFileResolverService: download file by got url failed url: %s : non 2xx status code")
	}

	return io.ReadAll(resp.Body)
}

func NewTelegramFileResolverService(fileGetter BotFileGetter) TelegramFileResolverService {
	return &TelegramFileResolverServiceImpl{
		fileGetter: fileGetter,
		log:        slog.With("service", "TelegramFileResolverService"),
	}
}
