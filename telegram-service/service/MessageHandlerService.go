package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/telegram-service/conf"
	"github.com/weoses/memelo/telegram-service/entity"
)

type MessageHandlerService interface {
	ProcessImageMessage(ctx context.Context, message *tgbotapi.Message) (*MessageHandlerResponse, error)
	ProcessVideoMessage(ctx context.Context, message *tgbotapi.Message) (*MessageHandlerResponse, error)
	ProcessDocumentMessage(ctx context.Context, message *tgbotapi.Message) (*MessageHandlerResponse, error)
	ProcessYouTubeMessage(ctx context.Context, message *tgbotapi.Message) (*MessageHandlerResponse, error)
}

const (
	parseMode    = "Markdown"
	msgForbidden = "You are not allowed to perform this action."
)

type MessageHandlerResponse struct {
	Message   string
	ParseMode string
}

type MessageHandlerServiceImpl struct {
	storage           StorageConnector
	fileResolver      TelegramFileResolverService
	permissionService PermissionService
	tmpDataService    commonservice.TmpDataService
	youtubeConnector  YouTubeConnector
	slogger           *slog.Logger
	staticAccountId   uuid.UUID
}

func (m MessageHandlerServiceImpl) ProcessImageMessage(ctx context.Context, message *tgbotapi.Message) (*MessageHandlerResponse, error) {
	if message.From == nil {
		return nil, errors.New("messageHandlerService: message has no sender")
	}
	return InvokeWithPermission(ctx, m.permissionService, message.From.ID, PermissionCreate, func() (*MessageHandlerResponse, error) {
		var fileId string
		if len(message.Photo) >= 1 {
			fileId = message.Photo[len(message.Photo)-1].FileID
		}
		if fileId == "" {
			return nil, errors.New("messageHandlerService: message dont contain image")
		}

		result, err := m.createMediaMeme(ctx, "image", fileId, m.staticAccountId)
		if err != nil {
			return nil, err
		}

		m.slogger.InfoContext(ctx, "meme created",
			"imageId", result.Id,
			"duplicate", result.DuplicateStatus)

		return &MessageHandlerResponse{
			Message: fmt.Sprintf(
				" Text: ```\n%s\n```\n"+
					" Tags: ```%s```\n"+
					" Caption: `%s`\n"+
					" ID: `%s` \n"+
					" Status: `%s`",
				result.Text,
				strings.Join(result.Tags, ", "),
				result.Caption,
				result.Id,
				result.DuplicateStatus),
			ParseMode: parseMode,
		}, nil
	})
}

func (m MessageHandlerServiceImpl) ProcessVideoMessage(ctx context.Context, message *tgbotapi.Message) (*MessageHandlerResponse, error) {
	if message.From == nil {
		return nil, errors.New("messageHandlerService: message has no sender")
	}
	return InvokeWithPermission(ctx, m.permissionService, message.From.ID, PermissionCreate, func() (*MessageHandlerResponse, error) {
		if message.Video == nil {
			return nil, errors.New("messageHandlerService: message does not contain a video")
		}

		result, err := m.createMediaMeme(ctx, "video", message.Video.FileID, m.staticAccountId)
		if err != nil {
			return nil, err
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
			ParseMode: parseMode,
		}, nil
	})
}

func (m MessageHandlerServiceImpl) ProcessDocumentMessage(ctx context.Context, message *tgbotapi.Message) (*MessageHandlerResponse, error) {
	if message.From == nil {
		return nil, errors.New("messageHandlerService: message has no sender")
	}
	return InvokeWithPermission(ctx, m.permissionService, message.From.ID, PermissionCreate, func() (*MessageHandlerResponse, error) {
		if message.Document == nil {
			return nil, errors.New("messageHandlerService: message does not contain a document")
		}

		mimeType := message.Document.MimeType
		if mimeType == "" {
			return nil, errors.New("messageHandlerService: message has no mime type")
		}

		var typ string
		if strings.HasPrefix(mimeType, "video/") {
			typ = "video"
		} else if strings.HasPrefix(mimeType, "image/") {
			typ = "image"
		} else {
			return nil, errors.New("messageHandlerService: invalid mime type")
		}

		result, err := m.createMediaMeme(ctx, typ, message.Document.FileID, m.staticAccountId)
		if err != nil {
			return nil, err
		}

		m.slogger.InfoContext(ctx, "document meme created",
			"memeId", result.Id,
			"duplicate", result.DuplicateStatus)

		return &MessageHandlerResponse{
			Message: fmt.Sprintf(
				" Text: ```\n%s\n```\n"+
					" Tags: ```%s```\n"+
					" Caption: `%s`\n"+
					" ID: `%s` \n"+
					" Status: `%s`",
				result.Text,
				strings.Join(result.Tags, ", "),
				result.Caption,
				result.Id,
				result.DuplicateStatus),
			ParseMode: parseMode,
		}, nil
	})
}

func (m MessageHandlerServiceImpl) ProcessYouTubeMessage(ctx context.Context, message *tgbotapi.Message) (*MessageHandlerResponse, error) {
	if message.From == nil {
		return nil, errors.New("messageHandlerService: message has no sender")
	}
	return InvokeWithPermission(ctx, m.permissionService, message.From.ID, PermissionCreate, func() (*MessageHandlerResponse, error) {
		jobId, err := m.youtubeConnector.DownloadVideoAsync(ctx, message.Text)
		if err != nil {
			return nil, fmt.Errorf("messageHandlerService: YouTube async job creation failed: %w", err)
		}

		var jobStatus *entity.YouTubeJobStatus
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
	poll:
		for {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-ticker.C:
				jobStatus, err = m.youtubeConnector.GetDownloadJobStatus(ctx, jobId)
				if err != nil {
					return nil, fmt.Errorf("messageHandlerService: GetDownloadJobStatus failed: %w", err)
				}
				switch jobStatus.State {
				case "done":
					break poll
				case "failed":
					return nil, fmt.Errorf("messageHandlerService: YouTube download failed: %s", jobStatus.Error)
				}
			}
		}
		if jobStatus.MimeType != "video/mp4" {
			return nil, fmt.Errorf("messageHandlerService: YouTube download failed: unexpected mime type: %s", jobStatus.MimeType)
		}

		s3Media, err := m.tmpDataService.WrapS3Path(ctx, jobStatus.S3Path)

		result, err := m.storage.CreateMeme(ctx, s3Media, "video", m.staticAccountId)
		if err != nil {
			return nil, fmt.Errorf("messageHandlerService: CreateMeme failed: %w", err)
		}

		m.slogger.InfoContext(ctx, "youtube meme created",
			"memeId", result.Id,
			"duplicate", result.DuplicateStatus)

		return &MessageHandlerResponse{
			Message: fmt.Sprintf(
				"\n```Text\n%s\n```\n ID: `%s` \n Status: `%s`\n Tags: ```%s```",
				result.Text,
				result.Id,
				result.DuplicateStatus,
				strings.Join(result.Tags, ", ")),
			ParseMode: parseMode,
		}, nil
	})
}

func (m MessageHandlerServiceImpl) createMediaMeme(ctx context.Context, reqType string, fileId string, accountId uuid.UUID) (*entity.MemeCreateResult, error) {
	fileURL, err := m.fileResolver.GetFileURL(ctx, fileId)
	if err != nil {
		return nil, fmt.Errorf("messageHandlerService: GetFileURL failed, fileId: %s : %w", fileId, err)
	}

	s3data, err := m.downloadToS3(ctx, fileURL)
	if err != nil {
		return nil, fmt.Errorf("messageHandlerService: downloadToS3 failed: %w", err)
	}
	defer helper.QuietClose(s3data, m.slogger)

	result, err := m.storage.CreateMeme(ctx, s3data, reqType, accountId)
	if err != nil {
		return nil, fmt.Errorf("messageHandlerService: CreateMeme failed : %w", err)
	}

	return result, nil
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
	permissionService PermissionService,
	tmpDataService commonservice.TmpDataService,
	youtubeConnector YouTubeConnector,
	cfg *conf.Config,
) MessageHandlerService {
	staticAccountId := uuid.MustParse(cfg.UserAccount.StaticUuid)
	return &MessageHandlerServiceImpl{
		storage:           storage,
		fileResolver:      fileResolver,
		permissionService: permissionService,
		tmpDataService:    tmpDataService,
		youtubeConnector:  youtubeConnector,
		slogger:           slog.With("service", "MessageHandlerService"),
		staticAccountId:   staticAccountId,
	}
}
