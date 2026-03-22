package observability

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

func TraceLogAttrs(ctx context.Context) []slog.Attr {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return nil
	}

	return []slog.Attr{
		slog.String("trace_id", spanContext.TraceID().String()),
		slog.String("span_id", spanContext.SpanID().String()),
	}
}
