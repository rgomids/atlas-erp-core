package usecases

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	billingports "github.com/rgomids/atlas-erp-core/internal/billing/public"
	paymentports "github.com/rgomids/atlas-erp-core/internal/payments/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/repositories"
	paymentevents "github.com/rgomids/atlas-erp-core/internal/payments/public/events"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

type paymentRepositoryFake struct {
	byID             map[string]entities.Payment
	byInvoice        map[string][]string
	byBillingAttempt map[string]string
}

var _ repositories.PaymentRepository = (*paymentRepositoryFake)(nil)

func newPaymentRepositoryFake() *paymentRepositoryFake {
	return &paymentRepositoryFake{
		byID:             map[string]entities.Payment{},
		byInvoice:        map[string][]string{},
		byBillingAttempt: map[string]string{},
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
	if _, exists := repository.byBillingAttempt[billingAttemptKey(payment.BillingID(), payment.AttemptNumber())]; exists {
		return entities.ErrPaymentAlreadyExists
	}

	repository.store(payment)
	return nil
}

func (repository *paymentRepositoryFake) Update(_ context.Context, payment entities.Payment) error {
	if _, exists := repository.byID[payment.ID()]; !exists {
		return entities.ErrPaymentNotFound
	}

	repository.store(payment)
	return nil
}

func (repository *paymentRepositoryFake) GetByID(_ context.Context, paymentID string) (entities.Payment, error) {
	payment, exists := repository.byID[paymentID]
	if !exists {
		return entities.Payment{}, entities.ErrPaymentNotFound
	}

	return payment, nil
}

func (repository *paymentRepositoryFake) GetByBillingIDAndAttempt(_ context.Context, billingID string, attemptNumber int) (entities.Payment, error) {
	paymentID, exists := repository.byBillingAttempt[billingAttemptKey(billingID, attemptNumber)]
	if !exists {
		return entities.Payment{}, entities.ErrPaymentNotFound
	}

	return repository.byID[paymentID], nil
}

func (repository *paymentRepositoryFake) ListByInvoiceID(_ context.Context, invoiceID string) ([]entities.Payment, error) {
	var payments []entities.Payment
	for _, paymentID := range repository.byInvoice[invoiceID] {
		payments = append(payments, repository.byID[paymentID])
	}

	return payments, nil
}

func (repository *paymentRepositoryFake) store(payment entities.Payment) {
	repository.byID[payment.ID()] = payment
	repository.byBillingAttempt[billingAttemptKey(payment.BillingID(), payment.AttemptNumber())] = payment.ID()

	existingIDs := repository.byInvoice[payment.InvoiceID()]
	for _, existingID := range existingIDs {
		if existingID == payment.ID() {
			return
		}
	}

	repository.byInvoice[payment.InvoiceID()] = append(repository.byInvoice[payment.InvoiceID()], payment.ID())
}

func billingAttemptKey(billingID string, attemptNumber int) string {
	return fmt.Sprintf("%s#%d", billingID, attemptNumber)
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
	delay  time.Duration
	calls  int
}

func (gateway *gatewayFake) Process(ctx context.Context, _ paymentports.GatewayRequest) (paymentports.GatewayResult, error) {
	gateway.calls++

	if gateway.delay > 0 {
		timer := time.NewTimer(gateway.delay)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return paymentports.GatewayResult{}, ctx.Err()
		case <-timer.C:
		}
	}

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

	sharedevent.Subscribe(bus, paymentevents.EventNamePaymentApproved, "test", sharedevent.HandlerFunc(func(_ context.Context, event sharedevent.Event) error {
		approvedEvents = append(approvedEvents, event.(paymentevents.PaymentApproved))
		return nil
	}))

	gateway := &gatewayFake{result: paymentports.GatewayResult{Status: "Approved", GatewayReference: "mock-approved"}}
	processBillingRequest := NewProcessBillingRequest(
		repository,
		gateway,
		txManagerFake{},
		bus,
		time.Second,
	)
	processBillingRequest.now = func() time.Time { return time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC) }

	payment, err := processBillingRequest.Execute(context.Background(), ProcessBillingRequestInput{
		BillingID:     "a4a40fd7-50ac-43c6-b6d1-a98ee0952603",
		InvoiceID:     "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
		CustomerID:    "88b6c2b1-f835-42b7-a06c-fd2f1a455faa",
		AmountCents:   4500,
		DueDate:       time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		AttemptNumber: 1,
	})
	if err != nil {
		t.Fatalf("process billing request: %v", err)
	}

	if payment.Status != "Approved" {
		t.Fatalf("expected approved payment, got %q", payment.Status)
	}

	if payment.AttemptNumber != 1 {
		t.Fatalf("expected attempt number 1, got %d", payment.AttemptNumber)
	}

	if len(approvedEvents) != 1 {
		t.Fatalf("expected 1 approved event, got %d", len(approvedEvents))
	}

	if approvedEvents[0].Payload.IdempotencyKey == "" {
		t.Fatal("expected approved event to carry idempotency key")
	}
}

func TestProcessBillingRequestReturnsExistingPaymentForDuplicateAttempt(t *testing.T) {
	t.Parallel()

	repository := newPaymentRepositoryFake()
	existingPayment, err := entities.RehydratePayment(
		"payment-id",
		"a4a40fd7-50ac-43c6-b6d1-a98ee0952603",
		"e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
		1,
		"billing:a4a40fd7-50ac-43c6-b6d1-a98ee0952603:attempt:1",
		"Approved",
		"mock-approved",
		"",
		time.Now().Add(-time.Minute),
		time.Now(),
	)
	if err != nil {
		t.Fatalf("rehydrate payment: %v", err)
	}
	if err := repository.Save(context.Background(), existingPayment); err != nil {
		t.Fatalf("seed approved payment: %v", err)
	}

	gateway := &gatewayFake{result: paymentports.GatewayResult{Status: "Approved", GatewayReference: "mock-approved"}}
	processBillingRequest := NewProcessBillingRequest(
		repository,
		gateway,
		txManagerFake{},
		sharedevent.NewSyncBus(),
		time.Second,
	)

	payment, err := processBillingRequest.Execute(context.Background(), ProcessBillingRequestInput{
		BillingID:     "a4a40fd7-50ac-43c6-b6d1-a98ee0952603",
		InvoiceID:     "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
		CustomerID:    "88b6c2b1-f835-42b7-a06c-fd2f1a455faa",
		AmountCents:   4500,
		DueDate:       time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		AttemptNumber: 1,
	})
	if err != nil {
		t.Fatalf("process duplicate billing request: %v", err)
	}

	if gateway.calls != 0 {
		t.Fatalf("expected duplicate attempt to skip gateway call, got %d calls", gateway.calls)
	}

	if payment.ID != "payment-id" {
		t.Fatalf("expected to return existing payment, got %q", payment.ID)
	}
}

func TestProcessBillingRequestRejectsApprovedPaymentForAnotherAttempt(t *testing.T) {
	t.Parallel()

	repository := newPaymentRepositoryFake()
	approvedPayment, err := entities.RehydratePayment(
		"payment-id",
		"a4a40fd7-50ac-43c6-b6d1-a98ee0952603",
		"e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
		1,
		"billing:a4a40fd7-50ac-43c6-b6d1-a98ee0952603:attempt:1",
		"Approved",
		"mock-approved",
		"",
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
		&gatewayFake{result: paymentports.GatewayResult{Status: "Approved", GatewayReference: "mock-approved"}},
		txManagerFake{},
		sharedevent.NewSyncBus(),
		time.Second,
	)

	_, err = processBillingRequest.Execute(context.Background(), ProcessBillingRequestInput{
		BillingID:     "a4a40fd7-50ac-43c6-b6d1-a98ee0952603",
		InvoiceID:     "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
		CustomerID:    "88b6c2b1-f835-42b7-a06c-fd2f1a455faa",
		AmountCents:   4500,
		DueDate:       time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		AttemptNumber: 2,
	})
	if !errors.Is(err, entities.ErrPaymentAlreadyExists) {
		t.Fatalf("expected duplicate approved payment error, got %v", err)
	}
}

func TestProcessBillingRequestPersistsGatewayFailureAndPublishesEvent(t *testing.T) {
	t.Parallel()

	repository := newPaymentRepositoryFake()
	bus := sharedevent.NewSyncBus()
	var failedEvents []paymentevents.PaymentFailed

	sharedevent.Subscribe(bus, paymentevents.EventNamePaymentFailed, "test", sharedevent.HandlerFunc(func(_ context.Context, event sharedevent.Event) error {
		failedEvents = append(failedEvents, event.(paymentevents.PaymentFailed))
		return nil
	}))

	processBillingRequest := NewProcessBillingRequest(
		repository,
		&gatewayFake{result: paymentports.GatewayResult{Status: "Failed", GatewayReference: "mock-failed"}},
		txManagerFake{},
		bus,
		time.Second,
	)
	processBillingRequest.now = func() time.Time { return time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC) }

	payment, err := processBillingRequest.Execute(context.Background(), ProcessBillingRequestInput{
		BillingID:     "a4a40fd7-50ac-43c6-b6d1-a98ee0952603",
		InvoiceID:     "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
		CustomerID:    "88b6c2b1-f835-42b7-a06c-fd2f1a455faa",
		AmountCents:   4500,
		DueDate:       time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		AttemptNumber: 1,
	})
	if err != nil {
		t.Fatalf("process failed billing request: %v", err)
	}

	if payment.Status != "Failed" {
		t.Fatalf("expected failed payment, got %q", payment.Status)
	}

	if payment.FailureCategory != string(entities.FailureCategoryGatewayDeclined) {
		t.Fatalf("expected gateway_declined failure category, got %q", payment.FailureCategory)
	}

	if len(failedEvents) != 1 {
		t.Fatalf("expected 1 failed event, got %d", len(failedEvents))
	}
}

func TestProcessBillingRequestClassifiesGatewayTimeoutAsFailedAttempt(t *testing.T) {
	t.Parallel()

	repository := newPaymentRepositoryFake()
	gateway := &gatewayFake{delay: 20 * time.Millisecond}
	processBillingRequest := NewProcessBillingRequest(
		repository,
		gateway,
		txManagerFake{},
		sharedevent.NewSyncBus(),
		5*time.Millisecond,
	)

	payment, err := processBillingRequest.Execute(context.Background(), ProcessBillingRequestInput{
		BillingID:     "a4a40fd7-50ac-43c6-b6d1-a98ee0952603",
		InvoiceID:     "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
		CustomerID:    "88b6c2b1-f835-42b7-a06c-fd2f1a455faa",
		AmountCents:   4500,
		DueDate:       time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		AttemptNumber: 1,
	})
	if err != nil {
		t.Fatalf("process timeout billing request: %v", err)
	}

	if payment.Status != "Failed" {
		t.Fatalf("expected failed payment after timeout, got %q", payment.Status)
	}

	if payment.FailureCategory != string(entities.FailureCategoryGatewayTimeout) {
		t.Fatalf("expected gateway_timeout failure category, got %q", payment.FailureCategory)
	}
}

func TestProcessPaymentAllowsManualRetryAfterFailure(t *testing.T) {
	t.Parallel()

	repository := newPaymentRepositoryFake()
	failedPayment, err := entities.RehydratePayment(
		"payment-failed",
		"a4a40fd7-50ac-43c6-b6d1-a98ee0952603",
		"e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
		1,
		"billing:a4a40fd7-50ac-43c6-b6d1-a98ee0952603:attempt:1",
		"Failed",
		"mock-failed",
		string(entities.FailureCategoryGatewayDeclined),
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
			ID:            "a4a40fd7-50ac-43c6-b6d1-a98ee0952603",
			InvoiceID:     "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
			CustomerID:    "88b6c2b1-f835-42b7-a06c-fd2f1a455faa",
			AmountCents:   4500,
			DueDate:       time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
			Status:        "Requested",
			AttemptNumber: 2,
		},
	}
	processBillingRequest := NewProcessBillingRequest(
		repository,
		&gatewayFake{result: paymentports.GatewayResult{Status: "Approved", GatewayReference: "mock-approved"}},
		txManagerFake{},
		sharedevent.NewSyncBus(),
		time.Second,
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

	if payment.AttemptNumber != 2 {
		t.Fatalf("expected retry attempt number 2, got %d", payment.AttemptNumber)
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
