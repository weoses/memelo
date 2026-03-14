package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"github.com/weoses/memelo/storage-service/key"
	"github.com/weoses/memelo/storage-service/service"
)

type SearchServiceApi struct {
	crud    service.MemeCrudService
	slogger *slog.Logger
}

func (api *SearchServiceApi) DeleteMeme(ctx context.Context, request *v1.DeleteMemeRequest) (*v1.DeleteMemeResponse, error) {
	ctx = context.WithValue(ctx, key.AccountId, request.AccountId)

	api.slogger.InfoContext(ctx, "DeleteMeme request", "id", request.Id)

	accountIdUuid, err := uuid.Parse(request.AccountId)
	if err != nil {
		return nil, fmt.Errorf("error parsing AccountId: %w", err)
	}

	idUuid, err := uuid.Parse(request.Id)
	if err != nil {
		return nil, fmt.Errorf("error parsing Id: %w", err)
	}

	if err = api.crud.DeleteMeme(ctx, accountIdUuid, idUuid); err != nil {
		api.slogger.ErrorContext(ctx, "DeleteMeme error", "id", request.Id, "err", err)
		return nil, err
	}

	api.slogger.InfoContext(ctx, "DeleteMeme success", "id", request.Id)

	return &v1.DeleteMemeResponse{Id: request.Id}, nil
}

func (api *SearchServiceApi) metadataToMemeDto(urls *service.MetadataWithUrls) *v1.MemeDto {
	dto := &v1.MemeDto{
		Id:        urls.Metadata.ImageId.String(),
		OcrResult: urls.Metadata.Result,
		ImageOriginal: &v1.ImageDto{
			Url: urls.UrlOriginal,
		},
		ImageThumbnail: &v1.ImageDto{
			Url:    urls.UrlThumb,
			Width:  int32(urls.Metadata.ThumbSize.Width),
			Height: int32(urls.Metadata.ThumbSize.Height),
		},
	}

	if urls.Metadata.ImageSize != nil {
		dto.ImageOriginal.Width = int32(urls.Metadata.ImageSize.Width)
		dto.ImageOriginal.Height = int32(urls.Metadata.ImageSize.Height)
	}

	return dto
}

func (api *SearchServiceApi) SearchMeme(ctx context.Context, req *v1.SearchMemeRequest) (*v1.SearchMemeResponse, error) {
	ctx = context.WithValue(ctx, key.AccountId, req.AccountId)

	api.slogger.InfoContext(ctx, "SearchMeme request", "query", req.Query)

	accountIdUuid, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, fmt.Errorf("error parsing AccountId: %w", err)
	}

	var afterIdUuid *uuid.UUID
	var pageSize *int

	if req.PageAfterId != nil {
		afterIdUuid_, err := uuid.Parse(*req.PageAfterId)
		if err != nil {
			return nil, fmt.Errorf("error parsing PageAfterId: %w", err)
		}
		afterIdUuid = &afterIdUuid_
	}

	if req.PageSize != nil {
		pageSize_ := int(*req.PageSize)
		pageSize = &pageSize_
	}

	data, err := api.crud.SearchMeme(ctx, accountIdUuid, req.Query, afterIdUuid, pageSize)
	if err != nil {
		api.slogger.ErrorContext(ctx, "SearchMeme error", "query", req.Query)
		return nil, err
	}

	api.slogger.Info("SearchMeme response", "count", len(data.Result))

	return &v1.SearchMemeResponse{
		Results: helper.TransformSlice(
			data.Result,
			make([]*v1.MemeDto, len(data.Result)),
			api.metadataToMemeDto),
		SearcherName: data.SearcherName,
	}, nil
}

func (api *SearchServiceApi) CreateMeme(ctx context.Context, req *v1.CreateMemeRequest) (*v1.CreateMemeResponse, error) {
	ctx = context.WithValue(ctx, key.AccountId, req.AccountId)

	api.slogger.InfoContext(ctx, "CreateMeme request")

	accountIdUuid, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, fmt.Errorf("error parsing AccountId: %w", err)
	}

	meme, err := api.crud.CreateMeme(ctx, accountIdUuid, req.RawImage)
	if err != nil {
		api.slogger.ErrorContext(ctx, "CreateMeme error", "err", err)
		return nil, err
	}

	api.slogger.Info("CreateMeme response",
		"id", meme.Metadata.Metadata.ImageId.String(),
		"status", meme.Status)

	return &v1.CreateMemeResponse{
		Result: api.metadataToMemeDto(meme.Metadata),
		Status: v1.CreateMemeStatus(meme.Status),
	}, nil
}

func (api *SearchServiceApi) GetMeme(context.Context, *v1.GetMemeRequest) (*v1.GetMemeResponse, error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("proto.memelo.v1.SearchService.GetMeme is not implemented"))
}

func NewSearchServiceApi(crud service.MemeCrudService) v1connect.SearchServiceHandler {
	return &SearchServiceApi{
		crud:    crud,
		slogger: slog.With("service", "SearchServiceApi"),
	}
}
