package observability

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type queryTraceContextKey string

const queryTraceKey queryTraceContextKey = "atlas_postgres_query_trace"

type queryTraceState struct {
	span      trace.Span
	startedAt time.Time
	operation string
	table     string
}

type queryTracer struct {
	runtime *Runtime
}

func newQueryTracer(runtime *Runtime) pgx.QueryTracer {
	if runtime == nil {
		return nil
	}

	return queryTracer{runtime: runtime}
}

func (tracer queryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	operation, table := sanitizeSQL(data.SQL)
	attrs := []attribute.KeyValue{
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", operation),
		attribute.String("db.sql.table", table),
	}

	ctx, span := tracer.runtime.Tracer("atlas-erp-core/postgres").Start(
		ctx,
		fmt.Sprintf("db.query %s %s", operation, table),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)

	return context.WithValue(ctx, queryTraceKey, queryTraceState{
		span:      span,
		startedAt: time.Now(),
		operation: operation,
		table:     table,
	})
}

func (tracer queryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	state, ok := ctx.Value(queryTraceKey).(queryTraceState)
	if !ok {
		return
	}

	tracer.runtime.RecordDBQuery(ctx, state.operation, state.table, time.Since(state.startedAt))
	tracer.runtime.CompleteSpan(state.span, data.Err, ErrorTypeInfrastructure)
}

func sanitizeSQL(statement string) (string, string) {
	tokens := strings.Fields(strings.ToUpper(statement))
	if len(tokens) == 0 {
		return "query", "unknown"
	}

	normalizedTokens := strings.Fields(statement)
	operation := strings.ToLower(tokens[0])

	switch operation {
	case "select", "delete":
		return operation, tableAfterKeyword(tokens, normalizedTokens, "FROM")
	case "insert":
		return operation, tableAfterKeyword(tokens, normalizedTokens, "INTO")
	case "update":
		if len(normalizedTokens) < 2 {
			return operation, "unknown"
		}

		return operation, normalizeIdentifier(normalizedTokens[1])
	default:
		return "query", "unknown"
	}
}

func tableAfterKeyword(upperTokens []string, rawTokens []string, keyword string) string {
	for index, token := range upperTokens {
		if token == keyword && index+1 < len(rawTokens) {
			return normalizeIdentifier(rawTokens[index+1])
		}
	}

	return "unknown"
}

func normalizeIdentifier(identifier string) string {
	trimmed := strings.Trim(identifier, "\"`[](),")
	if trimmed == "" {
		return "unknown"
	}

	return strings.ToLower(trimmed)
}
