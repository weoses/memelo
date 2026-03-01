package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	v1 "github.com/weoses/memelo/gen/proto/v1"
)

type SearchServiceApi struct {
	crud    MemeCrudService
	slogger *slog.Logger
}

func (api *SearchServiceApi) metadataToMemeDto(urls *MetadataWithUrls) *v1.MemeDto {
	return &v1.MemeDto{
		Id:        urls.Metadata.ImageId.String(),
		OcrResult: urls.Metadata.Result,
		ImageOriginal: &v1.ImageDto{
			Url:    urls.UrlOriginal,
			Width:  int32(urls.Metadata.ImageSize.Width),
			Height: int32(urls.Metadata.ImageSize.Height),
		},
		ImageThumbnail: &v1.ImageDto{
			Url:    urls.UrlThumb,
			Width:  int32(urls.Metadata.ThumbSize.Width),
			Height: int32(urls.Metadata.ThumbSize.Height),
		},
	}
}

func (api *SearchServiceApi) SearchMeme(ctx context.Context, req *v1.SearchMemeRequest) (*v1.SearchMemeResponse, error) {
	ctx = context.WithValue(ctx, "accountId", req.AccountId)

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

	api.slogger.Info("SearchMeme response", "count", len(data))

	return &v1.SearchMemeResponse{
		Results: helper.TransformSlice(
			data,
			make([]*v1.MemeDto, len(data)),
			api.metadataToMemeDto),
	}, nil
}

func (api *SearchServiceApi) CreateMeme(ctx context.Context, req *v1.CreateMemeRequest) (*v1.CreateMemeResponse, error) {
	ctx = context.WithValue(ctx, "accountId", req.AccountId)

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

func NewSearchServiceApi(crud MemeCrudService) *SearchServiceApi {
	return &SearchServiceApi{
		crud:    crud,
		slogger: slog.With("service", "SearchServiceApi"),
	}
}
