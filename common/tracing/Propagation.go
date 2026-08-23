package tracing

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// ExtractHTTP returns baseCtx carrying whatever remote span context the
// globally configured propagator (see InitTracer -- W3C traceparent/
// tracestate) finds in header. baseCtx, not a request's own r.Context()/
// c.Request().Context(), is the deliberate parameter: a caller whose real
// processing happens after the HTTP handler returns (e.g. telegram-service
// queuing an update for async dispatch) must build on its own long-lived
// context, not one that's cancelled the instant the handler returns.
func ExtractHTTP(baseCtx context.Context, header http.Header) context.Context {
	return otel.GetTextMapPropagator().Extract(baseCtx, propagation.HeaderCarrier(header))
}

// StartHTTP extracts any propagated parent span context from header (see
// ExtractHTTP), then starts and returns a new server-kind span as its
// child -- or as a new root span if header carried no valid traceparent,
// the common case for a public entry point (Telegram's servers, a
// browser) that never sends one.
func StartHTTP(baseCtx context.Context, tracerName, spanName string, header http.Header) (context.Context, trace.Span) {
	ctx := ExtractHTTP(baseCtx, header)
	return otel.Tracer(tracerName).Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
}
