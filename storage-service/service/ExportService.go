package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
	storage2 "github.com/weoses/memelo/storage-service/storage"
)

const exportPageSize = 100

type ExportService interface {
	// Export streams a "ready to pack" dtos of all exists images in database with metadata
	Export(ctx context.Context,
		accountId *uuid.UUID,
		id *uuid.UUID,
		callback func(ctx context.Context, items []ExportItem) error,
	) error
}

type ExportItem struct {
	Metadata       *entity.ElasticImageMetaData
	ImageOriginal  []byte
	ImageThumbnail []byte
}

type ExportServiceImpl struct {
	imageStorageService    storage2.ImageStorageService
	metadataStorageService storage2.MetadataStorageService
	slogger                *slog.Logger
}

func (e *ExportServiceImpl) Export(
	ctx context.Context,
	accountId *uuid.UUID,
	id *uuid.UUID,
	callback func(ctx context.Context, items []ExportItem) error,
) error {

	pageSize := exportPageSize
	var afterId *uuid.UUID
	processed := 0

	for {
		page, err := e.metadataStorageService.List(ctx, accountId, id, afterId, &pageSize)
		if err != nil {
			return fmt.Errorf("export: query metadata page failed: %w", err)
		}
		if len(page) == 0 {
			break
		}
		items := make([]ExportItem, len(page))

		for i, meta := range page {
			e.slogger.DebugContext(ctx, "export: processing item", "imageId", meta.ImageId)
			item := ExportItem{}
			// Original image
			origBytes, err := e.imageStorageService.GetImageBytes(ctx, meta.S3Id)
			if err != nil {
				return fmt.Errorf("export: fetch original image %s failed: %w", meta.ImageId, err)
			}
			item.ImageOriginal = origBytes

			// CreateThumbnail
			thumbBytes, err := e.imageStorageService.GetImageThumbBytes(ctx, meta.S3Id)
			if err != nil {
				return fmt.Errorf("export: fetch thumbnail %s failed: %w", meta.ImageId, err)
			}
			item.ImageThumbnail = thumbBytes

			// Metadata JSON → metadata/{imageId}.json
			item.Metadata = meta
			items[i] = item
			processed++
		}

		last := page[len(page)-1]
		afterId = &last.ImageId

		e.slogger.DebugContext(ctx, "export: invoke callback", "Processed", processed)
		err = callback(ctx, items)
		if err != nil {
			return fmt.Errorf("export: callback failed: %w", err)
		}

		// Last page — no more data
		if len(page) < pageSize {
			break
		}
	}

	e.slogger.InfoContext(ctx, "export: completed", "Processed", processed)
	return nil
}

func NewExportService(imageStore storage2.ImageStorageService, metadataStore storage2.MetadataStorageService) ExportService {
	return &ExportServiceImpl{
		imageStorageService:    imageStore,
		metadataStorageService: metadataStore,
		slogger:                slog.With("service", "ExportService"),
	}
}
