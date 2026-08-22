package tracing

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
)

const traceHeader = "X-Cloud-Trace-Context"

// NewInterceptor injects the incoming request's Cloud Trace header into
// ctx (see WithTrace) so every context-aware log call downstream
// (slog.InfoContext etc) is automatically trace-correlated. Put it first
// in connect.WithInterceptors(...) so the modified ctx is visible to
// interceptors after it (e.g. a logging interceptor).
func NewInterceptor(projectId string) connect.Interceptor {
	return &interceptor{projectId: projectId}
}

type interceptor struct {
	projectId string
}

func (i *interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		ctx = WithTrace(ctx, i.projectId, req.Header().Get(traceHeader))
		return next(ctx, req)
	}
}

func (i *interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		ctx = WithTrace(ctx, i.projectId, conn.RequestHeader().Get(traceHeader))
		return next(ctx, conn)
	}
}

// WrapStreamingClient is a no-op -- this interceptor is server-side only.
func (i *interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// HTTPMiddleware is the same trace-context injection for a plain
// net/http.Handler (services that don't route everything through Connect,
// e.g. telegram-service's /webhook).
func HTTPMiddleware(projectId string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := WithTrace(r.Context(), projectId, r.Header.Get(traceHeader))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
