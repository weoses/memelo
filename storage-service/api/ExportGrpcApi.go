package api

import (
	"context"
	"fmt"
	"log/slog"

	connect "connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	service2 "github.com/weoses/memelo/storage-service/service"
)

type ExportServiceApi struct {
	slogger *slog.Logger
	service service2.ExportService
}

func (e ExportServiceApi) ExportImages(ctx context.Context, request *v1.ExportRequest, c *connect.ServerStream[v1.ExportResponseChunk]) error {
	var accountId *uuid.UUID
	var id *uuid.UUID
	var err error

	accountId, err = helper.AccountIdUuid(request)
	if err != nil {
		return fmt.Errorf("account id uuid: %w", err)
	}

	id, err = helper.IdUuid(request)
	if err != nil {
		return fmt.Errorf("account id uuid: %w", err)
	}

	e.slogger.InfoContext(ctx, "Export: started")

	sent := 0
	callback := func(ctx context.Context, items []service2.ExportItem) error {
		for _, item := range items {

			chunk := v1.ExportResponseChunk{
				OcrResult: item.Metadata.Result,
				OriginalImage: &v1.ExportImageDto{
					Data: item.ImageOriginal,
				},
				ThumbnailImage: &v1.ExportImageDto{
					Data: item.ImageThumbnail,
				},
			}

			if item.Metadata.EmbeddingV1 != nil {
				chunk.Embedding = &v1.ExportImageEmbedding{
					Data:  *item.Metadata.EmbeddingV1.Data,
					Model: item.Metadata.EmbeddingV1.Model,
				}
			}

			if item.Metadata.ImageSize != nil {
				chunk.OriginalImage.Width = int32(item.Metadata.ImageSize.Width)
				chunk.OriginalImage.Height = int32(item.Metadata.ImageSize.Height)
			}

			if item.Metadata.ThumbSize != nil {
				chunk.ThumbnailImage.Width = int32(item.Metadata.ThumbSize.Width)
				chunk.ThumbnailImage.Height = int32(item.Metadata.ThumbSize.Height)
			}

			err := c.Send(&chunk)
			if err != nil {
				return fmt.Errorf("send chunk failed: %w", err)
			}

			sent++
			if sent%10 == 0 {
				e.slogger.DebugContext(ctx, "export: progress", "sent", sent)
			}
		}

		return nil
	}

	err = e.service.Export(ctx, accountId, id, callback)
	if err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	e.slogger.Info("export finished", "sent", sent)
	return nil
}

func NewExportServiceApi(service service2.ExportService) v1connect.ExportServiceHandler {
	return &ExportServiceApi{
		service: service,
		slogger: slog.With("service", "SearchServiceApi"),
	}
}
