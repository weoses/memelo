package middleware

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"
)

type loggingInterceptor struct {
	log *slog.Logger
}

// NewLoggingInterceptor returns a Connect server interceptor that logs every
// unary and streaming RPC with procedure, peer, connect code, and duration.
// Drop it into connect.WithInterceptors(...) on any handler.
func NewLoggingInterceptor(logger *slog.Logger) connect.Interceptor {
	return &loggingInterceptor{log: logger}
}

func (l *loggingInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		start := time.Now()
		procedure := req.Spec().Procedure
		peer := req.Peer().Addr

		l.log.InfoContext(ctx, "rpc request", "procedure", procedure, "peer", peer)

		resp, err := next(ctx, req)

		attrs := []any{
			"procedure", procedure,
			"peer", peer,
			"code", connect.CodeOf(err),
			"duration", time.Since(start).String(),
		}
		if err != nil {
			l.log.ErrorContext(ctx, "rpc error", append(attrs, "error", err)...)
		} else {
			l.log.InfoContext(ctx, "rpc response", attrs...)
		}

		return resp, err
	}
}

func (l *loggingInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		start := time.Now()
		procedure := conn.Spec().Procedure
		peer := conn.Peer().Addr

		l.log.InfoContext(ctx, "rpc stream open", "procedure", procedure, "peer", peer)

		err := next(ctx, conn)

		attrs := []any{
			"procedure", procedure,
			"peer", peer,
			"code", connect.CodeOf(err),
			"duration", time.Since(start).String(),
		}
		if err != nil {
			l.log.ErrorContext(ctx, "rpc stream error", append(attrs, "error", err)...)
		} else {
			l.log.InfoContext(ctx, "rpc stream closed", attrs...)
		}

		return err
	}
}

// WrapStreamingClient is a no-op — this interceptor is server-side only.
func (l *loggingInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}
