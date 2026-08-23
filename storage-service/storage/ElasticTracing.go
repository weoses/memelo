package storage

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("storage-service")

// traceElastic wraps a typed Elasticsearch request's Do method in a span
// named "elastic.<operation>" -- do is typically a bound method value like
// searchRequest.Do, whose signature already matches this generic shape.
func traceElastic[T any](ctx context.Context, operation string, index string, do func(ctx context.Context) (T, error)) (T, error) {
	ctx, span := tracer.Start(ctx, "elastic."+operation, trace.WithAttributes(
		attribute.String("db.system", "elasticsearch"),
		attribute.String("elastic.index", index),
	))
	defer span.End()

	result, err := do(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return result, err
}
