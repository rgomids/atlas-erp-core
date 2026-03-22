package observability

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewExportsRecordedMetrics(t *testing.T) {
	t.Parallel()

	runtime, err := New(context.Background(), Config{
		ServiceName: "atlas-erp-core",
		Environment: "test",
	})
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}

	runtime.RecordHTTPRequest(context.Background(), http.MethodPost, "/invoices", http.StatusCreated, "invoices", "", 0)
	runtime.RecordEventPublished(context.Background(), "InvoiceCreated", "invoices")
	runtime.RecordDBQuery(context.Background(), "insert", "invoices", 0)
	runtime.RecordGatewayRequest(context.Background(), 0)
	runtime.RecordGatewayFailure(context.Background(), ErrorTypeIntegration)
	runtime.RecordPaymentRetry(context.Background())

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	response := httptest.NewRecorder()
	runtime.MetricsHandler().ServeHTTP(response, request)

	body := response.Body.String()
	for _, fragment := range []string{
		"atlas_erp_http_requests_total",
		"atlas_erp_events_published_total",
		"atlas_erp_db_query_duration_seconds",
		"atlas_erp_gateway_failures_total",
		"atlas_erp_payment_retries_total",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected metrics output to contain %s, got %s", fragment, body)
		}
	}
}

func TestTraceLogAttrsIncludesActiveTraceContext(t *testing.T) {
	t.Parallel()

	exporter := tracetest.NewInMemoryExporter()
	runtime, err := New(context.Background(), Config{
		ServiceName:   "atlas-erp-core",
		Environment:   "test",
		TraceExporter: exporter,
	})
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}

	ctx, span := runtime.StartUseCase(context.Background(), "payments", "ProcessPayment")
	defer span.End()

	attrs := TraceLogAttrs(ctx)
	if len(attrs) != 2 {
		t.Fatalf("expected trace_id and span_id attrs, got %d", len(attrs))
	}

	buffer := &bytes.Buffer{}
	for _, attr := range attrs {
		buffer.WriteString(attr.String())
	}

	if !strings.Contains(buffer.String(), "trace_id") || !strings.Contains(buffer.String(), "span_id") {
		t.Fatalf("expected trace attrs to include trace_id and span_id, got %s", buffer.String())
	}
}
