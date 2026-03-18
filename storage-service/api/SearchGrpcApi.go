package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/weoses/memelo/common/helper"
	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/common/temp"

	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/key"
	"github.com/weoses/memelo/storage-service/service"
)

type SearchServiceApi struct {
	crud        service.MemeCrudService
	dataService commonservice.TmpDataService
	slogger     *slog.Logger
}

func (api *SearchServiceApi) DeleteAll(ctx context.Context, request *v1.DeleteAllRequest) (*v1.DeleteAllResponse, error) {
	ctx = context.WithValue(ctx, key.AccountId, request.AccountId)

	api.slogger.InfoContext(ctx, "DeleteAll request", "accountId", request.AccountId)

	accountIdUuid, err := uuid.Parse(request.AccountId)
	if err != nil {
		return nil, fmt.Errorf("error parsing AccountId: %w", err)
	}

	if err = api.crud.DeleteAll(ctx, accountIdUuid); err != nil {
		api.slogger.ErrorContext(ctx, "DeleteAll error", "accountId", request.AccountId, "err", err)
		return nil, err
	}

	api.slogger.InfoContext(ctx, "DeleteAll success", "accountId", request.AccountId)

	return &v1.DeleteAllResponse{}, nil
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
		MediaOriginal: &v1.ImageDto{
			Url: urls.UrlOriginal,
		},
		ImageThumbnail: &v1.ImageDto{
			Url:         urls.UrlThumb,
			ImageWidth:  helper.Addr(int32(urls.Metadata.ThumbSize.Width)),
			ImageHeight: helper.Addr(int32(urls.Metadata.ThumbSize.Height)),
		},
		Tags: urls.Metadata.Tags,
		Type: string(urls.Metadata.Type),
	}

	if urls.Metadata.ImageSize != nil {
		dto.MediaOriginal.ImageWidth = helper.Addr(int32(urls.Metadata.ImageSize.Width))
		dto.MediaOriginal.ImageHeight = helper.Addr(int32(urls.Metadata.ImageSize.Height))
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

	var meme *service.CreateResult
	var data temp.S3BackedData
	var metadataType entity.MetadataType
	if req.GetImage() != nil {
		metadataType = entity.ImageMetadataType
		data, err = api.toData(ctx, req.GetImage())
		if err != nil {
			return nil, fmt.Errorf("error reading image: %w", err)
		}
	} else if req.GetVideo() != nil {
		metadataType = entity.VideoMetadataType
		data, err = api.toData(ctx, req.GetVideo())
		if err != nil {
			return nil, fmt.Errorf("error reading video: %w", err)
		}
	}

	defer helper.QuietClose(data, api.slogger)
	meme, err = api.crud.CreateMeme(ctx, accountIdUuid, metadataType, data)

	defer helper.QuietClose(data, api.slogger)
	if err != nil || meme == nil {
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

func (api *SearchServiceApi) toData(ctx context.Context, media *v1.MediaDataDto) (temp.S3BackedData, error) {
	if media.GetS3Path() != "" {
		result, err := api.dataService.WrapS3Path(ctx, media.GetS3Path())
		if err != nil {
			return nil, fmt.Errorf("failed to create backed temp by s3 path: %w", err)
		}
		return result, nil
	}

	if media.GetData() != nil {
		data, err := api.dataService.ByBytes(ctx, media.GetData())
		if err != nil {
			return nil, fmt.Errorf("failed to get data from bytes: %w", err)
		}
		return data, nil
	}

	return nil, errors.New("media temp is empty")
}

func NewSearchServiceApi(crud service.MemeCrudService, dataService commonservice.TmpDataService) v1connect.SearchServiceHandler {
	return &SearchServiceApi{
		crud:        crud,
		dataService: dataService,

		slogger: slog.With("service", "SearchServiceApi"),
	}
}
