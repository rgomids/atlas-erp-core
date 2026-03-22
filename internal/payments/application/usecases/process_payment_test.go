package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	billingports "github.com/rgomids/atlas-erp-core/internal/billing/application/ports"
	paymentports "github.com/rgomids/atlas-erp-core/internal/payments/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
	paymentevents "github.com/rgomids/atlas-erp-core/internal/payments/domain/events"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/repositories"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

type paymentRepositoryFake struct {
	byID      map[string]entities.Payment
	byInvoice map[string][]string
}

var _ repositories.PaymentRepository = (*paymentRepositoryFake)(nil)

func newPaymentRepositoryFake() *paymentRepositoryFake {
	return &paymentRepositoryFake{
		byID:      map[string]entities.Payment{},
		byInvoice: map[string][]string{},
	}
}

func (repository *paymentRepositoryFake) HasApprovedByInvoiceID(_ context.Context, invoiceID string) (bool, error) {
	for _, paymentID := range repository.byInvoice[invoiceID] {
		if repository.byID[paymentID].Status() == entities.StatusApproved {
			return true, nil
		}
	}

	return false, nil
}

func (repository *paymentRepositoryFake) Save(_ context.Context, payment entities.Payment) error {
	repository.byID[payment.ID()] = payment
	repository.byInvoice[payment.InvoiceID()] = append(repository.byInvoice[payment.InvoiceID()], payment.ID())
	return nil
}

func (repository *paymentRepositoryFake) GetByID(_ context.Context, paymentID string) (entities.Payment, error) {
	payment, exists := repository.byID[paymentID]
	if !exists {
		return entities.Payment{}, entities.ErrInvalidPaymentID
	}

	return payment, nil
}

func (repository *paymentRepositoryFake) ListByInvoiceID(_ context.Context, invoiceID string) ([]entities.Payment, error) {
	var payments []entities.Payment
	for _, paymentID := range repository.byInvoice[invoiceID] {
		payments = append(payments, repository.byID[paymentID])
	}

	return payments, nil
}

type billingPortFake struct {
	snapshot billingports.BillingSnapshot
	err      error
	called   bool
}

func (port *billingPortFake) GetProcessableBillingByInvoiceID(context.Context, string) (billingports.BillingSnapshot, error) {
	port.called = true
	if port.err != nil {
		return billingports.BillingSnapshot{}, port.err
	}

	return port.snapshot, nil
}

type gatewayFake struct {
	result paymentports.GatewayResult
	err    error
}

func (gateway gatewayFake) Process(context.Context, paymentports.GatewayRequest) (paymentports.GatewayResult, error) {
	return gateway.result, gateway.err
}

type txManagerFake struct{}

func (txManagerFake) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestProcessBillingRequestApprovesAndPublishesEvent(t *testing.T) {
	t.Parallel()

	repository := newPaymentRepositoryFake()
	bus := sharedevent.NewSyncBus()
	var approvedEvents []paymentevents.PaymentApproved

	sharedevent.Subscribe(bus, paymentevents.PaymentApproved{}.Name(), "test", sharedevent.HandlerFunc(func(_ context.Context, event sharedevent.Event) error {
		approvedEvents = append(approvedEvents, event.(paymentevents.PaymentApproved))
		return nil
	}))

	processBillingRequest := NewProcessBillingRequest(
		repository,
		gatewayFake{result: paymentports.GatewayResult{Status: "Approved", GatewayReference: "mock-approved"}},
		txManagerFake{},
		bus,
	)
	processBillingRequest.now = func() time.Time { return time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC) }

	payment, err := processBillingRequest.Execute(context.Background(), ProcessBillingRequestInput{
		BillingID:   "a4a40fd7-50ac-43c6-b6d1-a98ee0952603",
		InvoiceID:   "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
		AmountCents: 4500,
		DueDate:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("process billing request: %v", err)
	}

	if payment.Status != "Approved" {
		t.Fatalf("expected approved payment, got %q", payment.Status)
	}

	if len(approvedEvents) != 1 {
		t.Fatalf("expected 1 approved event, got %d", len(approvedEvents))
	}

	if approvedEvents[0].InvoiceID != "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55" {
		t.Fatalf("expected invoice id to propagate, got %q", approvedEvents[0].InvoiceID)
	}
}

func TestProcessBillingRequestRejectsDuplicateApprovedPayment(t *testing.T) {
	t.Parallel()

	repository := newPaymentRepositoryFake()
	approvedPayment, err := entities.RehydratePayment(
		"payment-id",
		"billing-id",
		"e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
		"Approved",
		"mock-approved",
		time.Now().Add(-time.Minute),
		time.Now(),
	)
	if err != nil {
		t.Fatalf("rehydrate payment: %v", err)
	}
	if err := repository.Save(context.Background(), approvedPayment); err != nil {
		t.Fatalf("seed approved payment: %v", err)
	}

	processBillingRequest := NewProcessBillingRequest(
		repository,
		gatewayFake{result: paymentports.GatewayResult{Status: "Approved", GatewayReference: "mock-approved"}},
		txManagerFake{},
		sharedevent.NewSyncBus(),
	)

	_, err = processBillingRequest.Execute(context.Background(), ProcessBillingRequestInput{
		BillingID:   "a4a40fd7-50ac-43c6-b6d1-a98ee0952603",
		InvoiceID:   "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
		AmountCents: 4500,
		DueDate:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, entities.ErrPaymentAlreadyExists) {
		t.Fatalf("expected duplicate approved payment error, got %v", err)
	}
}

func TestProcessBillingRequestPersistsFailureAndPublishesEvent(t *testing.T) {
	t.Parallel()

	repository := newPaymentRepositoryFake()
	bus := sharedevent.NewSyncBus()
	var failedEvents []paymentevents.PaymentFailed

	sharedevent.Subscribe(bus, paymentevents.PaymentFailed{}.Name(), "test", sharedevent.HandlerFunc(func(_ context.Context, event sharedevent.Event) error {
		failedEvents = append(failedEvents, event.(paymentevents.PaymentFailed))
		return nil
	}))

	processBillingRequest := NewProcessBillingRequest(
		repository,
		gatewayFake{result: paymentports.GatewayResult{Status: "Failed", GatewayReference: "mock-failed"}},
		txManagerFake{},
		bus,
	)
	processBillingRequest.now = func() time.Time { return time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC) }

	payment, err := processBillingRequest.Execute(context.Background(), ProcessBillingRequestInput{
		BillingID:   "a4a40fd7-50ac-43c6-b6d1-a98ee0952603",
		InvoiceID:   "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
		AmountCents: 4500,
		DueDate:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("process failed billing request: %v", err)
	}

	if payment.Status != "Failed" {
		t.Fatalf("expected failed payment, got %q", payment.Status)
	}

	if len(failedEvents) != 1 {
		t.Fatalf("expected 1 failed event, got %d", len(failedEvents))
	}

	payments, err := repository.ListByInvoiceID(context.Background(), "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55")
	if err != nil {
		t.Fatalf("list payments by invoice: %v", err)
	}

	if len(payments) != 1 {
		t.Fatalf("expected 1 stored failed payment, got %d", len(payments))
	}
}

func TestProcessPaymentAllowsManualRetryAfterFailure(t *testing.T) {
	t.Parallel()

	repository := newPaymentRepositoryFake()
	failedPayment, err := entities.RehydratePayment(
		"payment-failed",
		"a4a40fd7-50ac-43c6-b6d1-a98ee0952603",
		"e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
		"Failed",
		"mock-failed",
		time.Now().Add(-2*time.Minute),
		time.Now().Add(-time.Minute),
	)
	if err != nil {
		t.Fatalf("rehydrate failed payment: %v", err)
	}
	if err := repository.Save(context.Background(), failedPayment); err != nil {
		t.Fatalf("seed failed payment: %v", err)
	}

	billingPort := &billingPortFake{
		snapshot: billingports.BillingSnapshot{
			ID:          "a4a40fd7-50ac-43c6-b6d1-a98ee0952603",
			InvoiceID:   "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
			AmountCents: 4500,
			DueDate:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
			Status:      "Requested",
		},
	}
	processBillingRequest := NewProcessBillingRequest(
		repository,
		gatewayFake{result: paymentports.GatewayResult{Status: "Approved", GatewayReference: "mock-approved"}},
		txManagerFake{},
		sharedevent.NewSyncBus(),
	)
	processPayment := NewProcessPayment(billingPort, processBillingRequest)

	payment, err := processPayment.Execute(context.Background(), ProcessPaymentInput{
		InvoiceID: "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
	})
	if err != nil {
		t.Fatalf("manual retry payment: %v", err)
	}

	if !billingPort.called {
		t.Fatal("expected billing compatibility port to be used")
	}

	if payment.Status != "Approved" {
		t.Fatalf("expected approved retry payment, got %q", payment.Status)
	}

	payments, err := repository.ListByInvoiceID(context.Background(), "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55")
	if err != nil {
		t.Fatalf("list payments by invoice: %v", err)
	}

	if len(payments) != 2 {
		t.Fatalf("expected two attempts after retry, got %d", len(payments))
	}
}

func TestProcessPaymentRejectsInvalidInvoiceID(t *testing.T) {
	t.Parallel()

	processPayment := NewProcessPayment(&billingPortFake{}, ProcessBillingRequest{})

	_, err := processPayment.Execute(context.Background(), ProcessPaymentInput{
		InvoiceID: "invalid",
	})
	if !errors.Is(err, entities.ErrInvalidInvoiceReference) {
		t.Fatalf("expected invalid invoice reference error, got %v", err)
	}
}
