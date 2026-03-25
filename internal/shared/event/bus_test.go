package event

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/rgomids/atlas-erp-core/internal/shared/logging"
)

type testEventPayload struct {
	InvoiceID string `json:"invoice_id"`
}

type testEvent struct {
	Envelope[testEventPayload]
}

func newTestEvent(ctx context.Context, eventName string, aggregateID string) testEvent {
	return testEvent{
		Envelope: NewEnvelope(ctx, eventName, aggregateID, Metadata{}.OccurredAt, testEventPayload{
			InvoiceID: aggregateID,
		}),
	}
}

func (event testEvent) Name() string {
	return event.Metadata.EventName
}

func TestSyncBusPublishesInSubscriptionOrder(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	logger, err := logging.NewWithWriter("info", buffer)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	previous := slog.Default()
	slog.SetDefault(logger)
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	bus := NewSyncBus()
	var executionOrder []string

	Subscribe(bus, "InvoiceCreated", "billing", HandlerFunc(func(context.Context, Event) error {
		executionOrder = append(executionOrder, "billing")
		return nil
	}))
	Subscribe(bus, "InvoiceCreated", "payments", HandlerFunc(func(context.Context, Event) error {
		executionOrder = append(executionOrder, "payments")
		return nil
	}))

	if err := Publish(context.Background(), bus, "invoices", newTestEvent(context.Background(), "InvoiceCreated", "invoice-123")); err != nil {
		t.Fatalf("publish event: %v", err)
	}

	expectedOrder := []string{"billing", "payments"}
	if !reflect.DeepEqual(executionOrder, expectedOrder) {
		t.Fatalf("expected order %v, got %v", expectedOrder, executionOrder)
	}

	for _, fragment := range []string{
		`"event":"InvoiceCreated"`,
		`"emitter_module":"invoices"`,
		`"consumer_module":"billing"`,
		`"consumer_module":"payments"`,
	} {
		if !strings.Contains(buffer.String(), fragment) {
			t.Fatalf("expected logs to contain %s, got %s", fragment, buffer.String())
		}
	}
}

func TestSyncBusStopsOnFirstHandlerError(t *testing.T) {
	t.Parallel()

	bus := NewSyncBus()
	expectedErr := errors.New("handler failed")
	executed := false

	Subscribe(bus, "BillingRequested", "payments", HandlerFunc(func(context.Context, Event) error {
		return expectedErr
	}))
	Subscribe(bus, "BillingRequested", "invoices", HandlerFunc(func(context.Context, Event) error {
		executed = true
		return nil
	}))

	err := Publish(context.Background(), bus, "billing", newTestEvent(context.Background(), "BillingRequested", "billing-123"))
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}

	if executed {
		t.Fatal("expected bus to stop after first handler error")
	}
}

func TestSyncBusDuplicatesConfiguredFirstEventDelivery(t *testing.T) {
	t.Parallel()

	bus := NewSyncBusWithOptions(SyncBusOptions{
		DuplicateFirstEventName: "BillingRequested",
	})

	deliveries := 0
	Subscribe(bus, "BillingRequested", "payments", HandlerFunc(func(context.Context, Event) error {
		deliveries++
		return nil
	}))

	if err := Publish(context.Background(), bus, "billing", newTestEvent(context.Background(), "BillingRequested", "billing-123")); err != nil {
		t.Fatalf("publish duplicated event: %v", err)
	}

	if deliveries != 2 {
		t.Fatalf("expected duplicated first delivery count 2, got %d", deliveries)
	}

	if err := Publish(context.Background(), bus, "billing", newTestEvent(context.Background(), "BillingRequested", "billing-456")); err != nil {
		t.Fatalf("publish second event: %v", err)
	}

	if deliveries != 3 {
		t.Fatalf("expected later deliveries to remain single, got %d", deliveries)
	}
}

func TestSyncBusInjectsFirstConsumerFailureForConfiguredModule(t *testing.T) {
	t.Parallel()

	bus := NewSyncBusWithOptions(SyncBusOptions{
		FailFirstConsumerEventName: "BillingRequested",
		FailFirstConsumerModule:    "payments",
	})

	paymentsExecuted := 0
	billingExecuted := 0

	Subscribe(bus, "BillingRequested", "payments", HandlerFunc(func(context.Context, Event) error {
		paymentsExecuted++
		return nil
	}))
	Subscribe(bus, "BillingRequested", "billing", HandlerFunc(func(context.Context, Event) error {
		billingExecuted++
		return nil
	}))

	err := Publish(context.Background(), bus, "billing", newTestEvent(context.Background(), "BillingRequested", "billing-123"))
	if !errors.Is(err, ErrInjectedConsumerFailure) {
		t.Fatalf("expected injected consumer failure, got %v", err)
	}

	if paymentsExecuted != 0 {
		t.Fatalf("expected payments handler to be skipped on injected failure, got %d executions", paymentsExecuted)
	}

	if billingExecuted != 0 {
		t.Fatalf("expected later handlers to stop after injected failure, got %d executions", billingExecuted)
	}

	if err := Publish(context.Background(), bus, "billing", newTestEvent(context.Background(), "BillingRequested", "billing-456")); err != nil {
		t.Fatalf("publish after injected failure: %v", err)
	}

	if paymentsExecuted != 1 {
		t.Fatalf("expected payments handler to recover on next delivery, got %d executions", paymentsExecuted)
	}

	if billingExecuted != 1 {
		t.Fatalf("expected later handlers to run after recovery, got %d executions", billingExecuted)
	}
}
