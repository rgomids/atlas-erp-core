package httpapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/rgomids/atlas-erp-core/internal/shared/logging"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
)

func TestRouterObservabilityUsesRoutePatternAndTraceContext(t *testing.T) {
	t.Parallel()

	logBuffer := &bytes.Buffer{}
	logger, err := logging.NewWithWriter("info", logBuffer)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	exporter := tracetest.NewInMemoryExporter()
	telemetry, err := observability.New(context.Background(), observability.Config{
		ServiceName:   "atlas-erp-core",
		Environment:   "test",
		TraceExporter: exporter,
	})
	if err != nil {
		t.Fatalf("create telemetry runtime: %v", err)
	}

	router := NewRouterWithObservability(logger, "X-Correlation-ID", telemetry, func(router chi.Router) {
		router.Get("/customers/{id}/invoices", func(writer http.ResponseWriter, request *http.Request) {
			WriteDomainError(writer, request, http.StatusNotFound, "invoice_not_found", "invoice was not found")
		})
	})

	traceID := trace.TraceID{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	parentSpanID := trace.SpanID{2, 2, 2, 2, 2, 2, 2, 2}
	request := httptest.NewRequest(http.MethodGet, "/customers/123e4567-e89b-12d3-a456-426614174000/invoices", nil)
	request.Header.Set("traceparent", fmt.Sprintf("00-%s-%s-01", traceID.String(), parentSpanID.String()))
	request.Header.Set("X-Correlation-ID", "req-http-observability")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}

	metricResponse := httptest.NewRecorder()
	metricRequest := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	router.ServeHTTP(metricResponse, metricRequest)

	metricBody := metricResponse.Body.String()
	for _, fragment := range []string{
		"atlas_erp_http_requests_total",
		"atlas_erp_http_request_errors_total",
		"/customers/{id}/invoices",
		"domain_error",
	} {
		if !strings.Contains(metricBody, fragment) {
			t.Fatalf("expected metrics output to contain %s, got %s", fragment, metricBody)
		}
	}

	logOutput := logBuffer.String()
	for _, fragment := range []string{
		`"route":"/customers/{id}/invoices"`,
		`"error_type":"domain_error"`,
		`"trace_id":"01010101010101010101010101010101"`,
	} {
		if !strings.Contains(logOutput, fragment) {
			t.Fatalf("expected log output to contain %s, got %s", fragment, logOutput)
		}
	}

	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected exported spans")
	}

	var found bool
	for _, span := range spans {
		if span.Name == "http.request GET /customers/{id}/invoices" {
			found = true
			if span.SpanContext.TraceID() != traceID {
				t.Fatalf("expected trace id %s, got %s", traceID.String(), span.SpanContext.TraceID().String())
			}

			if span.Parent.SpanID() != parentSpanID {
				t.Fatalf("expected parent span id %s, got %s", parentSpanID.String(), span.Parent.SpanID().String())
			}
		}
	}

	if !found {
		t.Fatalf("expected http span to be exported, got %#v", spans)
	}
}

func TestRouterObservabilityMarksInternalErrors(t *testing.T) {
	t.Parallel()

	logBuffer := &bytes.Buffer{}
	logger, err := logging.NewWithWriter("info", logBuffer)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	telemetry, err := observability.New(context.Background(), observability.Config{
		ServiceName: "atlas-erp-core",
		Environment: "test",
	})
	if err != nil {
		t.Fatalf("create telemetry runtime: %v", err)
	}

	router := NewRouterWithObservability(logger, "X-Correlation-ID", telemetry, func(router chi.Router) {
		router.Get("/customers/{id}", func(writer http.ResponseWriter, request *http.Request) {
			WriteInternalError(writer, request, errors.New("database unavailable"))
		})
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/customers/123e4567-e89b-12d3-a456-426614174000", nil)
	request.Header.Set("X-Correlation-ID", "req-http-internal")

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", response.Code)
	}

	metricResponse := httptest.NewRecorder()
	metricRequest := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	router.ServeHTTP(metricResponse, metricRequest)

	metricBody := metricResponse.Body.String()
	if !strings.Contains(metricBody, "infrastructure_error") {
		t.Fatalf("expected metrics output to contain infrastructure_error, got %s", metricBody)
	}

	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, `"error_type":"infrastructure_error"`) {
		t.Fatalf("expected log output to contain infrastructure_error, got %s", logOutput)
	}
}
