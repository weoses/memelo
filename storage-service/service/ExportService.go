package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/weoses/memelo/storage-service/entity"
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
	imageStore    ImageStorageService
	metadataStore MetadataStorageService
	slogger       *slog.Logger
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
		page, err := e.metadataStore.List(ctx, accountId, id, afterId, &pageSize)
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
			origBytes, err := e.imageStore.GetImageBytes(ctx, meta.S3Id)
			if err != nil {
				return fmt.Errorf("export: fetch original image %s failed: %w", meta.ImageId, err)
			}
			item.ImageOriginal = origBytes

			// CreateThumbnail
			thumbBytes, err := e.imageStore.GetImageThumbBytes(ctx, meta.S3Id)
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

		e.slogger.DebugContext(ctx, "export: invoke callback", "processed", processed)
		err = callback(ctx, items)
		if err != nil {
			return fmt.Errorf("export: callback failed: %w", err)
		}

		// Last page — no more data
		if len(page) < pageSize {
			break
		}
	}

	e.slogger.InfoContext(ctx, "export: completed", "processed", processed)
	return nil
}

func NewExportService(imageStore ImageStorageService, metadataStore MetadataStorageService) ExportService {
	return &ExportServiceImpl{
		imageStore:    imageStore,
		metadataStore: metadataStore,
		slogger:       slog.With("service", "ExportService"),
	}
}
