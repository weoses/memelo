package tracing

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
)

// appTraceparentHeader carries a second copy of the same W3C traceparent
// value as the standard "traceparent" header.
//
// Confirmed empirically: Cloud Run's own edge-layer (GFE) auto-tracing
// intercepts and rewrites the standard "traceparent" header in transit,
// replacing its parent-span-id with its own newly created edge span before
// forwarding to the app container -- and that edge span sometimes never
// gets exported to Cloud Trace (seen specifically when two requests land
// within ~60ms of each other on a reused connection), leaving a "missing
// span id" gap where the edge span should be. GFE only knows to rewrite
// the standard header name; a differently-named header passes through
// untouched as an ordinary application header. So extraction prefers this
// header when present -- our own application-level span tree then chains
// service-to-service span directly, staying fully connected end-to-end
// regardless of whether Cloud Run's own edge span made it into Cloud Trace.
const appTraceparentHeader = "X-App-Traceparent"

// dualTraceContextPropagator wraps propagation.TraceContext{}: Inject writes
// the standard "traceparent" header as usual, then duplicates its value onto
// appTraceparentHeader. Extract prefers appTraceparentHeader when present,
// falling back to the standard header otherwise (e.g. a request that never
// passed through one of our own services, like an inbound webhook).
type dualTraceContextPropagator struct {
	base propagation.TraceContext
}

func (p dualTraceContextPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	p.base.Inject(ctx, carrier)
	if tp := carrier.Get("traceparent"); tp != "" {
		carrier.Set(appTraceparentHeader, tp)
	}
}

func (p dualTraceContextPropagator) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	if custom := carrier.Get(appTraceparentHeader); custom != "" {
		return p.base.Extract(ctx, propagation.MapCarrier{"traceparent": custom})
	}
	return p.base.Extract(ctx, carrier)
}

func (p dualTraceContextPropagator) Fields() []string {
	return append(p.base.Fields(), appTraceparentHeader)
}
