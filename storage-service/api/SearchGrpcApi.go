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
	"github.com/weoses/memelo/storage-service/utils"

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

func (api *SearchServiceApi) UpdateMeme(ctx context.Context, req *v1.UpdateMemeRequest) (*v1.UpdateMemeResponse, error) {
	ctx = context.WithValue(ctx, key.AccountId, req.AccountId)

	accountId, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, fmt.Errorf("error parsing AccountId: %w", err)
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, fmt.Errorf("error parsing Id: %w", err)
	}

	var originalData temp.S3BackedData
	if req.Original != nil {
		var closeable bool
		originalData, closeable, err = api.toData(ctx, req.Original)
		if err != nil {
			return nil, fmt.Errorf("error reading original media: %w", err)
		}
		if closeable {
			defer helper.QuietClose(originalData, api.slogger)
		}
	}

	var thumbnailData temp.S3BackedData
	if req.Thumbnail != nil {
		var closeable bool
		thumbnailData, closeable, err = api.toData(ctx, req.Thumbnail)
		if err != nil {
			return nil, fmt.Errorf("error reading thumbnail: %w", err)
		}
		if closeable {
			defer helper.QuietClose(thumbnailData, api.slogger)
		}
	}

	result, err := api.crud.UpdateMeme(ctx, service.UpdateMemeInput{
		Id:              id,
		AccountId:       accountId,
		OnScreenText:    req.OnScreenText,
		AudioTranscript: req.AudioTranscription,
		Caption:         req.Caption,
		AudioTrack:      req.AudioTrack,
		Original:        originalData,
		Thumbnail:       thumbnailData,
	})
	if errors.Is(err, service.ErrMemeNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		api.slogger.ErrorContext(ctx, "UpdateMeme error", "id", req.Id, "err", err)
		return nil, err
	}

	return &v1.UpdateMemeResponse{Result: api.metadataToMemeDto(result)}, nil
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
	return utils.MetadataToMemeDto(urls)
}

func (api *SearchServiceApi) SearchMeme(ctx context.Context, req *v1.SearchMemeRequest) (*v1.SearchMemeResponse, error) {
	ctx = context.WithValue(ctx, key.AccountId, req.AccountId)

	api.slogger.InfoContext(ctx, "SearchMeme request", "query", req.Query)

	accountIdUuid, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, fmt.Errorf("error parsing AccountId: %w", err)
	}

	afterId := utils.PipelineAfterIDFromProto(req.AfterId)
	pageSize := int(req.PageSize)
	if pageSize == 0 {
		return nil, fmt.Errorf("invalid pageSize")
	}

	var metadataType *entity.MetadataType
	if t := req.GetType(); t != "" {
		mt := entity.MetadataType(t)
		metadataType = &mt
	}

	data, err := api.crud.SearchMeme(ctx, accountIdUuid, req.Query, metadataType, afterId, pageSize)
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
		LastId: utils.PipelineAfterIDToProto(data.AfterID),
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
	var closeable bool
	var metadataType entity.MetadataType
	if req.GetImage() != nil {
		metadataType = entity.ImageMetadataType
		data, closeable, err = api.toData(ctx, req.GetImage())
		if err != nil {
			return nil, fmt.Errorf("error reading image: %w", err)
		}
	} else if req.GetVideo() != nil {
		metadataType = entity.VideoMetadataType
		data, closeable, err = api.toData(ctx, req.GetVideo())
		if err != nil {
			return nil, fmt.Errorf("error reading video: %w", err)
		}
	}

	if closeable {
		defer helper.QuietClose(data, api.slogger)
	}

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

func (api *SearchServiceApi) GetMeme(ctx context.Context, req *v1.GetMemeRequest) (*v1.GetMemeResponse, error) {
	ctx = context.WithValue(ctx, key.AccountId, req.AccountId)

	accountId, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, fmt.Errorf("error parsing AccountId: %w", err)
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, fmt.Errorf("error parsing Id: %w", err)
	}

	api.slogger.InfoContext(ctx, "GetMeme request", "id", id)
	result, err := api.crud.GetMeme(ctx, accountId, id)
	if errors.Is(err, service.ErrMemeNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		api.slogger.ErrorContext(ctx, "GetMeme error", "id", req.Id, "err", err)
		return nil, err
	}

	return &v1.GetMemeResponse{Result: api.metadataToMemeDto(result)}, nil
}

func (api *SearchServiceApi) GetRandomMeme(ctx context.Context, req *v1.GetRandomMemeRequest) (*v1.GetRandomMemeResponse, error) {
	ctx = context.WithValue(ctx, key.AccountId, req.AccountId)

	accountId, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, fmt.Errorf("error parsing AccountId: %w", err)
	}

	mediaType := entity.MetadataType(req.GetType())

	api.slogger.InfoContext(ctx, "GetRandomMeme request", "accountId", req.AccountId, "type", req.GetType())
	result, err := api.crud.GetRandomMeme(ctx, accountId, mediaType)
	if errors.Is(err, service.ErrMemeNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		api.slogger.ErrorContext(ctx, "GetRandomMeme error", "err", err)
		return nil, err
	}

	return &v1.GetRandomMemeResponse{Result: api.metadataToMemeDto(result)}, nil
}

func (api *SearchServiceApi) toData(ctx context.Context, media *v1.MediaDataDto) (temp.S3BackedData, bool, error) {
	return utils.ToData(ctx, api.dataService, media)
}

func NewSearchServiceApi(crud service.MemeCrudService, dataService commonservice.TmpDataService) v1connect.SearchServiceHandler {
	return &SearchServiceApi{
		crud:        crud,
		dataService: dataService,

		slogger: slog.With("service", "SearchServiceApi"),
	}
}
