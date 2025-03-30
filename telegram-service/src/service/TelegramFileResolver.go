package service

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

type BotFileGetter interface {
	GetFileDirectURL(fileID string) (string, error)
}

type TelegramFileResolverService interface {
	GetFile(fileID string) ([]byte, error)
}

type TelegramFileResolverServiceImpl struct {
	fileGetter BotFileGetter
}

// GetFile implements TelegramFileResolverService.
func (t *TelegramFileResolverServiceImpl) GetFile(fileID string) ([]byte, error) {
	url, err := t.fileGetter.GetFileDirectURL(fileID)
	if err != nil {
		return nil, fmt.Errorf("TelegramFileResolverService: GetFileDirectURL failed, fileId: %s : %w", fileID, err)
	}
	resp, err := http.Get(url)
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
	}
}
