package event

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
)

type observedEvent struct {
	Envelope[observedEventPayload]
}

type observedEventPayload struct {
	InvoiceID     string `json:"invoice_id"`
	CustomerID    string `json:"customer_id"`
	AttemptNumber int    `json:"attempt_number"`
}

func newObservedEvent(ctx context.Context, eventName string) observedEvent {
	return observedEvent{
		Envelope: NewEnvelope(ctx, eventName, "invoice-123", Metadata{}.OccurredAt, observedEventPayload{
			InvoiceID:     "invoice-123",
			CustomerID:    "customer-456",
			AttemptNumber: 1,
		}),
	}
}

func (event observedEvent) Name() string {
	return event.Metadata.EventName
}

func TestSyncBusObservabilityExportsSpansAndMetrics(t *testing.T) {
	t.Parallel()

	exporter := tracetest.NewInMemoryExporter()
	telemetry, err := observability.New(context.Background(), observability.Config{
		ServiceName:   "atlas-erp-core",
		Environment:   "test",
		TraceExporter: exporter,
	})
	if err != nil {
		t.Fatalf("create telemetry runtime: %v", err)
	}

	bus := NewSyncBusWithObservability(telemetry)
	Subscribe(bus, "InvoiceCreated", "billing", HandlerFunc(func(context.Context, Event) error {
		return nil
	}))

	ctx, rootSpan := telemetry.StartUseCase(context.Background(), "invoices", "CreateInvoice")
	if err := Publish(ctx, bus, "invoices", newObservedEvent(ctx, "InvoiceCreated")); err != nil {
		t.Fatalf("publish observed event: %v", err)
	}
	rootSpan.End()

	spans := exporter.GetSpans()
	expectedNames := []string{
		"application.usecase invoices.CreateInvoice",
		"event.publish InvoiceCreated",
		"event.consume billing.InvoiceCreated",
	}
	for _, expectedName := range expectedNames {
		if !containsSpanNamed(spans, expectedName) {
			t.Fatalf("expected spans to contain %s, got %#v", expectedName, spans)
		}
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	telemetry.MetricsHandler().ServeHTTP(response, request)

	body := response.Body.String()
	for _, fragment := range []string{
		"atlas_erp_events_published_total",
		"atlas_erp_events_consumed_total",
		"InvoiceCreated",
		"invoices",
		"billing",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected metrics output to contain %s, got %s", fragment, body)
		}
	}
}

func containsSpanNamed(spans tracetest.SpanStubs, expectedName string) bool {
	for _, span := range spans {
		if span.Name == expectedName {
			return true
		}
	}

	return false
}
