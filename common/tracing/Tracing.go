// Package tracing correlates application logs with Cloud Run's
// per-request trace, so Cloud Console's "show entries for this trace"
// button (on a request log entry) surfaces the app's own log lines for
// that request alongside it.
package tracing

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

type traceCtxKey struct{}

// WithTrace parses the standard Cloud Trace request header
// ("TRACE_ID/SPAN_ID;o=TRACE_TRUE") and stores the fully-qualified trace
// resource name Cloud Logging's structured-log parser requires
// ("projects/<project>/traces/<TRACE_ID>") in ctx. A bare trace ID is not
// enough -- Cloud Logging only correlates the fully-qualified form.
//
// Returns ctx unchanged if projectId or header is empty, so this is a
// no-op wherever ProjectId isn't configured (local/docker-compose).
func WithTrace(ctx context.Context, projectId, header string) context.Context {
	if projectId == "" || header == "" {
		return ctx
	}
	traceID := header
	if i := strings.IndexByte(header, '/'); i >= 0 {
		traceID = header[:i]
	}
	if traceID == "" {
		return ctx
	}
	return context.WithValue(ctx, traceCtxKey{}, fmt.Sprintf("projects/%s/traces/%s", projectId, traceID))
}

func fromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(traceCtxKey{}).(string)
	return v, ok
}

// Handler wraps a slog.Handler, attaching a "logging.googleapis.com/trace"
// attribute (see WithTrace) to every record whose context carries one.
// Only meaningful for the JSON handler -- Cloud Logging's structured-log
// parser doesn't extract fields from plain text payloads.
type Handler struct {
	slog.Handler
}

func NewHandler(base slog.Handler) *Handler {
	return &Handler{Handler: base}
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if trace, ok := fromContext(ctx); ok {
		r.AddAttrs(slog.String("logging.googleapis.com/trace", trace))
	}
	return h.Handler.Handle(ctx, r)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{Handler: h.Handler.WithGroup(name)}
}
