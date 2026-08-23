package tracing

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// Handler wraps a slog.Handler, attaching Cloud Logging's structured-log
// trace-correlation fields (logging.googleapis.com/trace, /spanId,
// /trace_sampled) to every record whose context carries a valid span --
// i.e. every request that passed through an otelconnect interceptor or
// tracing.StartHTTP. No-op when ctx carries no valid span, or when
// projectId is empty (local/docker-compose), consistent with every other
// place ProjectId gates Cloud-Logging-specific behavior in this codebase.
type Handler struct {
	slog.Handler
	projectId string
}

func NewHandler(base slog.Handler, projectId string) *Handler {
	return &Handler{Handler: base, projectId: projectId}
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if h.projectId != "" {
		if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
			r.AddAttrs(
				slog.String("logging.googleapis.com/trace",
					fmt.Sprintf("projects/%s/traces/%s", h.projectId, sc.TraceID())),
				slog.String("logging.googleapis.com/spanId", sc.SpanID().String()),
				slog.Bool("logging.googleapis.com/trace_sampled", sc.IsSampled()),
			)
		}
	}
	return h.Handler.Handle(ctx, r)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{Handler: h.Handler.WithAttrs(attrs), projectId: h.projectId}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{Handler: h.Handler.WithGroup(name), projectId: h.projectId}
}
