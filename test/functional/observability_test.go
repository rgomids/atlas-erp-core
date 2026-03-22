package functional_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/rgomids/atlas-erp-core/internal/billing"
	"github.com/rgomids/atlas-erp-core/internal/customers"
	"github.com/rgomids/atlas-erp-core/internal/invoices"
	"github.com/rgomids/atlas-erp-core/internal/payments"
	paymentports "github.com/rgomids/atlas-erp-core/internal/payments/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/integration"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	httpapi "github.com/rgomids/atlas-erp-core/internal/shared/http"
	"github.com/rgomids/atlas-erp-core/internal/shared/logging"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
	"github.com/rgomids/atlas-erp-core/internal/shared/outbox"
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
	"github.com/rgomids/atlas-erp-core/test/support"
)

func TestPhase5HTTPPropagatesTraceparentAndExposesMetrics(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	exporter := tracetest.NewInMemoryExporter()
	telemetry, err := observability.New(ctx, observability.Config{
		ServiceName:   "atlas-erp-core",
		Environment:   "test",
		TraceExporter: exporter,
	})
	if err != nil {
		t.Fatalf("create telemetry runtime: %v", err)
	}

	pool, err := sharedpostgres.Open(ctx, databaseConfig, telemetry)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer pool.Close()

	logBuffer := &bytes.Buffer{}
	server := newObservedFunctionalServerWithTimeout(t, pool, integration.NewMockGateway(), logBuffer, time.Second, telemetry)
	defer server.Close()

	customerResponse := postJSON(t, server.URL+"/customers", `{"name":"Atlas Co","document":"12345678900","email":"team@atlas.io"}`)
	var customer struct {
		ID string `json:"id"`
	}
	decodeResponse(t, customerResponse, &customer)

	traceID := trace.TraceID{3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3}
	parentSpanID := trace.SpanID{4, 4, 4, 4, 4, 4, 4, 4}
	request, err := http.NewRequest(
		http.MethodPost,
		server.URL+"/invoices",
		strings.NewReader(`{"customer_id":"`+customer.ID+`","amount_cents":1599,"due_date":"2026-03-25"}`),
	)
	if err != nil {
		t.Fatalf("create invoice request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Correlation-ID", "req-phase5-functional-001")
	request.Header.Set("traceparent", fmt.Sprintf("00-%s-%s-01", traceID.String(), parentSpanID.String()))

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post invoice: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected invoice creation status 201, got %d", response.StatusCode)
	}

	metricsResponse, err := server.Client().Get(server.URL + "/metrics")
	if err != nil {
		t.Fatalf("get metrics: %v", err)
	}
	defer metricsResponse.Body.Close()

	metricsBody := readResponseBody(t, metricsResponse)
	for _, fragment := range []string{
		"atlas_erp_http_requests_total",
		"atlas_erp_events_published_total",
		"/invoices",
	} {
		if !strings.Contains(metricsBody, fragment) {
			t.Fatalf("expected metrics output to contain %s, got %s", fragment, metricsBody)
		}
	}

	logOutput := logBuffer.String()
	for _, fragment := range []string{
		`"trace_id":"03030303030303030303030303030303"`,
		`"request_id":"req-phase5-functional-001"`,
		`"event_name":"InvoiceCreated"`,
	} {
		if !strings.Contains(logOutput, fragment) {
			t.Fatalf("expected log output to contain %s, got %s", fragment, logOutput)
		}
	}

	spans := exporter.GetSpans()
	expectedSpans := []string{
		"http.request POST /invoices",
		"application.usecase invoices.CreateInvoice",
		"event.publish InvoiceCreated",
		"event.consume billing.InvoiceCreated",
		"integration.gateway payments.Process",
	}
	for _, expectedSpan := range expectedSpans {
		if !functionalContainsSpan(spans, expectedSpan, traceID) {
			t.Fatalf("expected spans to contain %s with trace id %s, got %#v", expectedSpan, traceID.String(), spans)
		}
	}
}

func TestPhase5HTTPLogsValidationErrorWithTraceContext(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	exporter := tracetest.NewInMemoryExporter()
	telemetry, err := observability.New(ctx, observability.Config{
		ServiceName:   "atlas-erp-core",
		Environment:   "test",
		TraceExporter: exporter,
	})
	if err != nil {
		t.Fatalf("create telemetry runtime: %v", err)
	}

	pool, err := sharedpostgres.Open(ctx, databaseConfig, telemetry)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer pool.Close()

	logBuffer := &bytes.Buffer{}
	server := newObservedFunctionalServerWithTimeout(t, pool, integration.NewMockGateway(), logBuffer, time.Second, telemetry)
	defer server.Close()

	traceID := trace.TraceID{5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5}
	parentSpanID := trace.SpanID{6, 6, 6, 6, 6, 6, 6, 6}
	request, err := http.NewRequest(http.MethodPost, server.URL+"/customers", strings.NewReader(`{"name":"Atlas Co"}`))
	if err != nil {
		t.Fatalf("create invalid customer request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Correlation-ID", "req-phase5-functional-002")
	request.Header.Set("traceparent", fmt.Sprintf("00-%s-%s-01", traceID.String(), parentSpanID.String()))

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post invalid customer: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected validation status 400, got %d", response.StatusCode)
	}

	logOutput := logBuffer.String()
	for _, fragment := range []string{
		`"trace_id":"05050505050505050505050505050505"`,
		`"request_id":"req-phase5-functional-002"`,
		`"error_type":"validation_error"`,
	} {
		if !strings.Contains(logOutput, fragment) {
			t.Fatalf("expected log output to contain %s, got %s", fragment, logOutput)
		}
	}
}

func newObservedFunctionalServer(t *testing.T, pool *pgxpool.Pool, gateway paymentports.PaymentGateway, logWriter *bytes.Buffer, telemetry *observability.Runtime) *httptest.Server {
	return newObservedFunctionalServerWithTimeout(t, pool, gateway, logWriter, time.Second, telemetry)
}

func newObservedFunctionalServerWithTimeout(t *testing.T, pool *pgxpool.Pool, gateway paymentports.PaymentGateway, logWriter *bytes.Buffer, timeout time.Duration, telemetry *observability.Runtime) *httptest.Server {
	t.Helper()

	logger, err := logging.NewWithWriter("info", logWriter)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	eventBus := sharedevent.NewSyncBusWithObservability(telemetry, outbox.NewPostgresRecorder(pool))
	customerModule := customers.NewModule(pool, eventBus, telemetry)
	invoiceModule := invoices.NewModule(pool, customerModule.ExistenceChecker(), eventBus, telemetry)
	billingModule := billing.NewModule(pool, eventBus, telemetry)
	paymentModule := payments.NewModule(pool, billingModule.PaymentPort(), eventBus, gateway, payments.ModuleConfig{
		GatewayTimeout: timeout,
	}, telemetry)

	return httptest.NewServer(httpapi.NewRouterWithObservability(
		logger,
		"X-Correlation-ID",
		telemetry,
		customerModule.Routes,
		invoiceModule.Routes,
		paymentModule.Routes,
	))
}

func readResponseBody(t *testing.T, response *http.Response) string {
	t.Helper()

	buffer := &bytes.Buffer{}
	if _, err := buffer.ReadFrom(response.Body); err != nil {
		t.Fatalf("read response body: %v", err)
	}

	return buffer.String()
}

func functionalContainsSpan(spans tracetest.SpanStubs, expectedName string, expectedTraceID trace.TraceID) bool {
	for _, span := range spans {
		if span.Name == expectedName && span.SpanContext.TraceID() == expectedTraceID {
			return true
		}
	}

	return false
}
