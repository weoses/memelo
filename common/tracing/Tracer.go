// Package tracing wires OpenTelemetry distributed tracing (W3C traceparent
// propagation via Connect RPC and plain HTTP entry points) and bridges the
// active span into Cloud Logging's trace-correlation fields.
package tracing

import (
	"context"
	"fmt"

	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// InitTracer installs the global W3C traceparent/tracestate propagator
// (always, even when export is disabled -- extracting/injecting a header
// costs nothing) and, when projectId is non-empty, a TracerProvider that
// batches spans to Google Cloud Trace.
//
// When projectId is empty (local/docker-compose default, same convention
// as LoggingConfig.ProjectId / RequireGoogleIDToken), no TracerProvider is
// installed: otel.GetTracerProvider() then returns the SDK's built-in
// no-op provider, so every otelconnect interceptor and every
// tracing.StartHTTP call becomes a free no-op -- spans are never
// allocated, sampled, or exported.
//
// Call once per service, as early as possible in main() (right after
// config.InitLogs). The returned shutdown func flushes any buffered spans
// and must run before process exit.
func InitTracer(ctx context.Context, serviceName, projectId string) (shutdown func(context.Context) error, err error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		dualTraceContextPropagator{},
		propagation.Baggage{},
	))

	noop := func(context.Context) error { return nil }
	if projectId == "" {
		return noop, nil
	}

	exporter, err := texporter.New(texporter.WithProjectID(projectId))
	if err != nil {
		return nil, fmt.Errorf("tracing.InitTracer: exporter: %w", err)
	}

	res, err := resource.Merge(resource.Default(), resource.NewSchemaless(
		attribute.String("service.name", serviceName),
	))
	if err != nil {
		return nil, fmt.Errorf("tracing.InitTracer: resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		// AlwaysSample: fine at this traffic volume -- debugging value
		// outweighs Cloud Trace's free-tier span quota risk. If that ever
		// changes, swap for sdktrace.TraceIDRatioBased(x); would need a
		// new config field (give it a config.yaml line in every service
		// or Viper's AllKeys()-env-override workaround silently no-ops
		// the override, same gotcha as RequireGoogleIDToken/Log.Format).
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}
