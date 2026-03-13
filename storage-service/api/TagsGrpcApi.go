package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"github.com/weoses/memelo/storage-service/service"
)

type TagsGrpcApiImpl struct {
	slogger    *slog.Logger
	tagService service.TagService
}

func (a *TagsGrpcApiImpl) CreateTag(ctx context.Context, req *v1.CreateTagRequest) (*v1.CreateTagResponse, error) {
	entity, err := a.tagService.CreateTag(ctx, req.Tag, req.Description)
	if err != nil {
		if errors.Is(err, service.ErrTagDuplicate) {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		return nil, fmt.Errorf("create tag failed: %w", err)
	}

	tag := &v1.Tag{
		Id:          entity.Id.String(),
		Tag:         entity.Tag,
		Description: entity.Description,
	}

	return &v1.CreateTagResponse{Result: tag}, nil
}

func (a *TagsGrpcApiImpl) DeleteTag(ctx context.Context, req *v1.DeleteTagRequest) (*v1.DeleteTagResponse, error) {
	parsedId, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid id: %w", err))
	}

	if err := a.tagService.DeleteTag(ctx, parsedId); err != nil {
		return nil, fmt.Errorf("delete tag failed: %w", err)
	}

	return &v1.DeleteTagResponse{}, nil
}

func (a *TagsGrpcApiImpl) ListTags(ctx context.Context, req *v1.ListTagRequest) (*v1.ListTagResponse, error) {
	entities, err := a.tagService.ListTags(ctx, req.TagName, req.TagDescription)
	if err != nil {
		return nil, fmt.Errorf("list tags failed: %w", err)
	}

	tags := make([]*v1.Tag, len(entities))
	for i, e := range entities {
		tags[i] = &v1.Tag{
			Id:          e.Id.String(),
			Tag:         e.Tag,
			Description: e.Description,
		}
	}

	return &v1.ListTagResponse{Result: tags}, nil
}

func NewTagsGrpcApi(tagService service.TagService) v1connect.TagsServiceHandler {
	return &TagsGrpcApiImpl{
		slogger:    slog.With("service", "TagsGrpcApi"),
		tagService: tagService,
	}
}
