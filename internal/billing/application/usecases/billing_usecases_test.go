package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	billingports "github.com/rgomids/atlas-erp-core/internal/billing/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/billing/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/billing/domain/repositories"
	billingevents "github.com/rgomids/atlas-erp-core/internal/billing/public/events"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

type billingRepositoryFake struct {
	byID      map[string]entities.Billing
	byInvoice map[string]string
}

var _ repositories.BillingRepository = (*billingRepositoryFake)(nil)

func newBillingRepositoryFake() *billingRepositoryFake {
	return &billingRepositoryFake{
		byID:      map[string]entities.Billing{},
		byInvoice: map[string]string{},
	}
}

func (repository *billingRepositoryFake) Save(_ context.Context, billing entities.Billing) error {
	if _, exists := repository.byInvoice[billing.InvoiceID()]; exists {
		return entities.ErrBillingAlreadyExists
	}

	repository.byID[billing.ID()] = billing
	repository.byInvoice[billing.InvoiceID()] = billing.ID()
	return nil
}

func (repository *billingRepositoryFake) GetByID(_ context.Context, billingID string) (entities.Billing, error) {
	billing, exists := repository.byID[billingID]
	if !exists {
		return entities.Billing{}, entities.ErrBillingNotFound
	}

	return billing, nil
}

func (repository *billingRepositoryFake) GetByInvoiceID(_ context.Context, invoiceID string) (entities.Billing, error) {
	billingID, exists := repository.byInvoice[invoiceID]
	if !exists {
		return entities.Billing{}, entities.ErrBillingNotFound
	}

	return repository.byID[billingID], nil
}

func (repository *billingRepositoryFake) GetByInvoiceIDForUpdate(ctx context.Context, invoiceID string) (entities.Billing, error) {
	return repository.GetByInvoiceID(ctx, invoiceID)
}

func (repository *billingRepositoryFake) Update(_ context.Context, billing entities.Billing) error {
	if _, exists := repository.byID[billing.ID()]; !exists {
		return entities.ErrBillingNotFound
	}

	repository.byID[billing.ID()] = billing
	repository.byInvoice[billing.InvoiceID()] = billing.ID()
	return nil
}

type txManagerFake struct{}

func (txManagerFake) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestCreateBillingFromInvoicePublishesBillingRequested(t *testing.T) {
	t.Parallel()

	repository := newBillingRepositoryFake()
	bus := sharedevent.NewSyncBus()
	var requestedEvents []billingevents.BillingRequested

	sharedevent.Subscribe(bus, billingevents.EventNameBillingRequested, "test", sharedevent.HandlerFunc(func(_ context.Context, event sharedevent.Event) error {
		requestedEvents = append(requestedEvents, event.(billingevents.BillingRequested))
		return nil
	}))

	createBilling := NewCreateBillingFromInvoice(repository, bus)
	createBilling.now = func() time.Time { return time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC) }

	billing, err := createBilling.Execute(context.Background(), CreateBillingFromInvoiceInput{
		InvoiceID:   "1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe",
		CustomerID:  "7adf3d42-7b1d-4d2b-a7d6-5d977b7576aa",
		AmountCents: 1599,
		DueDate:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create billing: %v", err)
	}

	if billing.Status != "Requested" {
		t.Fatalf("expected requested billing, got %q", billing.Status)
	}

	if len(requestedEvents) != 1 {
		t.Fatalf("expected 1 billing requested event, got %d", len(requestedEvents))
	}

	if requestedEvents[0].Payload.AttemptNumber != 1 {
		t.Fatalf("expected initial attempt number 1, got %d", requestedEvents[0].Payload.AttemptNumber)
	}

	if requestedEvents[0].EventMetadata().AggregateID != billing.ID {
		t.Fatalf("expected aggregate id %q, got %q", billing.ID, requestedEvents[0].EventMetadata().AggregateID)
	}
}

func TestCreateBillingFromInvoiceIsIdempotent(t *testing.T) {
	t.Parallel()

	repository := newBillingRepositoryFake()
	bus := sharedevent.NewSyncBus()
	eventCount := 0

	sharedevent.Subscribe(bus, billingevents.EventNameBillingRequested, "test", sharedevent.HandlerFunc(func(_ context.Context, event sharedevent.Event) error {
		eventCount++
		return nil
	}))

	createBilling := NewCreateBillingFromInvoice(repository, bus)
	input := CreateBillingFromInvoiceInput{
		InvoiceID:   "1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe",
		CustomerID:  "7adf3d42-7b1d-4d2b-a7d6-5d977b7576aa",
		AmountCents: 1599,
		DueDate:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
	}

	if _, err := createBilling.Execute(context.Background(), input); err != nil {
		t.Fatalf("create first billing: %v", err)
	}

	if _, err := createBilling.Execute(context.Background(), input); err != nil {
		t.Fatalf("create duplicate billing: %v", err)
	}

	if len(repository.byID) != 1 {
		t.Fatalf("expected a single stored billing, got %d", len(repository.byID))
	}

	if eventCount != 1 {
		t.Fatalf("expected a single billing requested event, got %d", eventCount)
	}
}

func TestGetProcessableBillingByInvoiceIDReactivatesFailedBilling(t *testing.T) {
	t.Parallel()

	repository := newBillingRepositoryFake()
	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	billing, err := entities.NewBilling(
		"billing-id",
		"1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe",
		"7adf3d42-7b1d-4d2b-a7d6-5d977b7576aa",
		1599,
		time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		now,
	)
	if err != nil {
		t.Fatalf("new billing: %v", err)
	}
	billing.MarkFailed(now.Add(time.Minute))
	if err := repository.Save(context.Background(), billing); err != nil {
		t.Fatalf("save billing: %v", err)
	}

	getProcessableBilling := NewGetProcessableBillingByInvoiceID(repository, txManagerFake{})
	getProcessableBilling.now = func() time.Time { return now.Add(2 * time.Minute) }

	processableBilling, err := getProcessableBilling.Execute(context.Background(), billing.InvoiceID())
	if err != nil {
		t.Fatalf("get processable billing: %v", err)
	}

	if processableBilling.Status != "Requested" {
		t.Fatalf("expected requested billing after reactivation, got %q", processableBilling.Status)
	}

	if processableBilling.AttemptNumber != 2 {
		t.Fatalf("expected retry attempt number 2, got %d", processableBilling.AttemptNumber)
	}
}

func TestGetProcessableBillingByInvoiceIDRejectsApprovedBilling(t *testing.T) {
	t.Parallel()

	repository := newBillingRepositoryFake()
	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	billing, err := entities.NewBilling(
		"billing-id",
		"1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe",
		"7adf3d42-7b1d-4d2b-a7d6-5d977b7576aa",
		1599,
		time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		now,
	)
	if err != nil {
		t.Fatalf("new billing: %v", err)
	}
	billing.MarkApproved(now.Add(time.Minute))
	if err := repository.Save(context.Background(), billing); err != nil {
		t.Fatalf("save billing: %v", err)
	}

	getProcessableBilling := NewGetProcessableBillingByInvoiceID(repository, txManagerFake{})

	_, err = getProcessableBilling.Execute(context.Background(), billing.InvoiceID())
	if !errors.Is(err, entities.ErrBillingAlreadyApproved) {
		t.Fatalf("expected billing already approved error, got %v", err)
	}
}

var _ billingports.TransactionManager = txManagerFake{}
