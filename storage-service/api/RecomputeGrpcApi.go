package api

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/weoses/memelo/common/helper"
	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
	"github.com/weoses/memelo/storage-service/service"
)

type RecomputeGrpcApiImpl struct {
	slogger          *slog.Logger
	recomputeService service.RecomputeService
}

func (r RecomputeGrpcApiImpl) RecomputeOcrData(ctx context.Context, request *v1.RecomputeRequest, c *connect.ServerStream[v1.RecomputeStatus]) error {
	accountId, err := helper.AccountIdUuid(request)
	if err != nil {
		return fmt.Errorf("account id uuid: %w", err)
	}

	id, err := helper.IdUuid(request)
	if err != nil {
		return fmt.Errorf("id uuid: %w", err)
	}
	callback := func(ctx context.Context, ps service.ProgressDataRecompute) error {
		return c.Send(&v1.RecomputeStatus{
			Processed: int32(ps.Processed),
		})
	}

	return r.recomputeService.Recompute(
		ctx,
		accountId,
		id,
		callback)
}

func NewRecomputeGrpcApi(service service.RecomputeService) v1connect.RecomputeServiceHandler {
	return &RecomputeGrpcApiImpl{
		slogger:          slog.With("service", "RecomputeGrpcApi"),
		recomputeService: service,
	}
}
