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

type testEvent struct {
	name string
}

func (event testEvent) Name() string {
	return event.name
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

	if err := Publish(context.Background(), bus, "invoices", testEvent{name: "InvoiceCreated"}); err != nil {
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

	err := Publish(context.Background(), bus, "billing", testEvent{name: "BillingRequested"})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}

	if executed {
		t.Fatal("expected bus to stop after first handler error")
	}
}
