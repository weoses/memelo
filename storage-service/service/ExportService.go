package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/storage-service/entity"
	storage2 "github.com/weoses/memelo/storage-service/storage"
)

const exportPageSize = 100

type ExportService interface {
	// Export streams a "ready to pack" dtos of all exists images in database with metadataService
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
	imageStorageService    storage2.MediaStorageService
	metadataStorageService storage2.MetadataStorageService
	slogger                *slog.Logger
}

func (e *ExportServiceImpl) Export(
	ctx context.Context,
	accountId *uuid.UUID,
	id *uuid.UUID,
	callback func(ctx context.Context, items []ExportItem) error,
) error {
	if id != nil {
		item, err := e.metadataStorageService.GetById(ctx, *accountId, *id)
		if err != nil {
			return fmt.Errorf("export: query metadataService item failed: %w", err)
		}
		if item == nil {
			return nil
		}
		exportedItem, err := e.exportOne(ctx, item)
		if err != nil {
			return fmt.Errorf("export: export item failed: %w", err)
		}
		return callback(ctx, []ExportItem{*exportedItem})
	}

	var sortKey entity.ElasticSortKey
	processed := 0

	for {
		var page []*entity.ElasticImageMetaData
		var nextKey entity.ElasticSortKey
		var err error

		if accountId != nil {
			page, nextKey, err = e.metadataStorageService.GetByAccountIdOrderByCreated(ctx, *accountId, nil, sortKey, exportPageSize)
		} else {
			page, nextKey, err = e.metadataStorageService.GetAll(ctx, sortKey, exportPageSize)
		}
		if err != nil {
			return fmt.Errorf("export: query metadataService page failed: %w", err)
		}
		if len(page) == 0 {
			break
		}

		items := make([]ExportItem, len(page))
		for i, meta := range page {
			item, err := e.exportOne(ctx, meta)
			if err != nil {
				return fmt.Errorf("export: export item failed: %w", err)
			}
			items[i] = *item
			processed++
		}

		e.slogger.DebugContext(ctx, "export: invoke callback", "Processed", processed)
		if err = callback(ctx, items); err != nil {
			return fmt.Errorf("export: callback failed: %w", err)
		}

		if len(page) < exportPageSize {
			break
		}
		sortKey = nextKey
	}

	e.slogger.InfoContext(ctx, "export: completed", "Processed", processed)
	return nil
}

func (e *ExportServiceImpl) exportOne(ctx context.Context, meta *entity.ElasticImageMetaData) (*ExportItem, error) {
	e.slogger.DebugContext(ctx, "export: processing item", "imageId", meta.ImageId)
	item := &ExportItem{}
	// Original image
	origData, err := e.imageStorageService.Read(ctx, meta.S3Id, storageMediaType(meta.Type, SavedOriginal))
	if err != nil {
		return nil, fmt.Errorf("export: fetch original image %s failed: %w", meta.ImageId, err)
	}
	defer helper.QuietClose(origData, e.slogger)
	item.ImageOriginal, err = origData.ReadAll()

	if err != nil {
		return nil, fmt.Errorf("export: read original image %s failed: %w", meta.ImageId, err)
	}

	// CreateThumbnail
	thumbData, err := e.imageStorageService.Read(ctx, meta.S3Id, storageMediaType(meta.Type, SavedThumb))
	if err != nil {
		return nil, fmt.Errorf("export: fetch thumbnail %s failed: %w", meta.ImageId, err)
	}
	defer helper.QuietClose(thumbData, e.slogger)

	item.ImageThumbnail, err = thumbData.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("export: read thumbnail %s failed: %w", meta.ImageId, err)
	}

	// Metadata JSON → metadataService/{imageId}.json
	item.Metadata = meta
	return item, nil
}

func NewExportService(imageStore storage2.MediaStorageService, metadataStore storage2.MetadataStorageService) ExportService {
	return &ExportServiceImpl{
		imageStorageService:    imageStore,
		metadataStorageService: metadataStore,
		slogger:                slog.With("service", "ExportService"),
	}
}
