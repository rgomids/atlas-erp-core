package event_test

import (
	"context"
	"testing"
	"time"

	billingevents "github.com/rgomids/atlas-erp-core/internal/billing/public/events"
	customerevents "github.com/rgomids/atlas-erp-core/internal/customers/public/events"
	invoiceevents "github.com/rgomids/atlas-erp-core/internal/invoices/public/events"
	paymentevents "github.com/rgomids/atlas-erp-core/internal/payments/public/events"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

func TestPublicEventCatalogsExposeUniqueDescriptors(t *testing.T) {
	t.Parallel()

	catalogs := [][]sharedevent.Descriptor{
		customerevents.Catalog(),
		invoiceevents.Catalog(),
		billingevents.Catalog(),
		paymentevents.Catalog(),
	}

	seen := map[string]sharedevent.Descriptor{}
	for _, catalog := range catalogs {
		for _, descriptor := range catalog {
			if descriptor.Name == "" {
				t.Fatal("event descriptor name must not be empty")
			}
			if descriptor.ProducerModule == "" {
				t.Fatal("event descriptor producer module must not be empty")
			}

			if existing, ok := seen[descriptor.Name]; ok {
				t.Fatalf("duplicate event descriptor %q between %s and %s", descriptor.Name, existing.ProducerModule, descriptor.ProducerModule)
			}

			seen[descriptor.Name] = descriptor
		}
	}

	expectedNames := []string{
		customerevents.EventNameCustomerCreated,
		invoiceevents.EventNameInvoiceCreated,
		invoiceevents.EventNameInvoicePaid,
		billingevents.EventNameBillingRequested,
		paymentevents.EventNamePaymentApproved,
		paymentevents.EventNamePaymentFailed,
	}
	for _, expectedName := range expectedNames {
		if _, ok := seen[expectedName]; !ok {
			t.Fatalf("expected event descriptor %q to exist", expectedName)
		}
	}
}

func TestPublicEventConstructorsPopulateMetadata(t *testing.T) {
	t.Parallel()

	occurredAt := time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC)

	testCases := []struct {
		name              string
		event             sharedevent.Event
		expectedName      string
		expectedAggregate string
	}{
		{
			name:              "customer created",
			event:             customerevents.NewCustomerCreated(context.Background(), "customer-1", occurredAt),
			expectedName:      customerevents.EventNameCustomerCreated,
			expectedAggregate: "customer-1",
		},
		{
			name:              "invoice created",
			event:             invoiceevents.NewInvoiceCreated(context.Background(), "invoice-1", "customer-1", 1599, occurredAt, occurredAt),
			expectedName:      invoiceevents.EventNameInvoiceCreated,
			expectedAggregate: "invoice-1",
		},
		{
			name:              "billing requested",
			event:             billingevents.NewBillingRequested(context.Background(), "billing-1", "invoice-1", "customer-1", 1599, occurredAt, 2, occurredAt),
			expectedName:      billingevents.EventNameBillingRequested,
			expectedAggregate: "billing-1",
		},
		{
			name:              "payment failed",
			event:             paymentevents.NewPaymentFailed(context.Background(), "payment-1", "billing-1", "invoice-1", "customer-1", 2, "billing:1:attempt:2", "gateway_timeout", "gw-timeout", occurredAt),
			expectedName:      paymentevents.EventNamePaymentFailed,
			expectedAggregate: "payment-1",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			metadata := testCase.event.EventMetadata()
			if metadata.EventID == "" {
				t.Fatal("expected event id to be populated")
			}
			if metadata.CorrelationID == "" {
				t.Fatal("expected correlation id to be populated")
			}
			if metadata.EventName != testCase.expectedName {
				t.Fatalf("expected event name %q, got %q", testCase.expectedName, metadata.EventName)
			}
			if metadata.AggregateID != testCase.expectedAggregate {
				t.Fatalf("expected aggregate id %q, got %q", testCase.expectedAggregate, metadata.AggregateID)
			}
			if !metadata.OccurredAt.Equal(occurredAt) {
				t.Fatalf("expected occurred_at %s, got %s", occurredAt, metadata.OccurredAt)
			}
		})
	}
}
