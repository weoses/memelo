package service

import (
	"context"
	"log/slog"

	connect "connectrpc.com/connect"
	v1 "github.com/weoses/memelo/gen/proto/v1"
	"github.com/weoses/memelo/gen/proto/v1/v1connect"
)

type RecomputeGrpcApiImpl struct {
	slogger *slog.Logger
}

func (r RecomputeGrpcApiImpl) RecomputeOcrData(ctx context.Context, request *v1.RecomputeRequest, c *connect.ServerStream[v1.RecomputeStatus]) error {

}

func NewRecomputeGrpcApi() v1connect.RecomputeServiceHandler {
	return &RecomputeGrpcApiImpl{
		slogger: slog.With("service", "RecomputeGrpcApi"),
	}
}
