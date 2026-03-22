package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	invoiceports "github.com/rgomids/atlas-erp-core/internal/invoices/application/ports"
	invoiceentities "github.com/rgomids/atlas-erp-core/internal/invoices/domain/entities"
	paymentports "github.com/rgomids/atlas-erp-core/internal/payments/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/repositories"
)

type paymentRepositoryFake struct {
	byID      map[string]entities.Payment
	byInvoice map[string]string
}

var _ repositories.PaymentRepository = (*paymentRepositoryFake)(nil)

func newPaymentRepositoryFake() *paymentRepositoryFake {
	return &paymentRepositoryFake{
		byID:      map[string]entities.Payment{},
		byInvoice: map[string]string{},
	}
}

func (repository *paymentRepositoryFake) ExistsByInvoiceID(_ context.Context, invoiceID string) (bool, error) {
	_, exists := repository.byInvoice[invoiceID]
	return exists, nil
}

func (repository *paymentRepositoryFake) Save(_ context.Context, payment entities.Payment) error {
	repository.byID[payment.ID()] = payment
	repository.byInvoice[payment.InvoiceID()] = payment.ID()
	return nil
}

func (repository *paymentRepositoryFake) GetByID(_ context.Context, paymentID string) (entities.Payment, error) {
	payment, exists := repository.byID[paymentID]
	if !exists {
		return entities.Payment{}, entities.ErrInvalidPaymentID
	}

	return payment, nil
}

type invoicePaymentPortFake struct {
	snapshot     invoiceports.InvoiceSnapshot
	getErr       error
	markedAsPaid bool
}

func (port *invoicePaymentPortFake) GetPayableInvoice(context.Context, string) (invoiceports.InvoiceSnapshot, error) {
	if port.getErr != nil {
		return invoiceports.InvoiceSnapshot{}, port.getErr
	}

	return port.snapshot, nil
}

func (port *invoicePaymentPortFake) MarkAsPaid(context.Context, string, time.Time) error {
	port.markedAsPaid = true
	return nil
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

func TestProcessPaymentApprovesAndMarksInvoicePaid(t *testing.T) {
	t.Parallel()

	repository := newPaymentRepositoryFake()
	invoicePort := &invoicePaymentPortFake{
		snapshot: invoiceports.InvoiceSnapshot{
			ID:          "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
			CustomerID:  "1d7349c1-0d5d-4fd6-9cfe-dcb12e6aa9f5",
			AmountCents: 4500,
			Status:      "Pending",
			DueDate:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		},
	}
	processPayment := NewProcessPayment(
		repository,
		invoicePort,
		gatewayFake{result: paymentports.GatewayResult{Status: "Approved", GatewayReference: "mock-approved"}},
		txManagerFake{},
	)
	processPayment.now = func() time.Time { return time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC) }

	payment, err := processPayment.Execute(context.Background(), ProcessPaymentInput{
		InvoiceID: "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
	})
	if err != nil {
		t.Fatalf("process payment: %v", err)
	}

	if payment.Status != "Approved" {
		t.Fatalf("expected approved payment, got %q", payment.Status)
	}

	if !invoicePort.markedAsPaid {
		t.Fatal("expected invoice to be marked as paid")
	}
}

func TestProcessPaymentRejectsDuplicateInvoicePayment(t *testing.T) {
	t.Parallel()

	repository := newPaymentRepositoryFake()
	repository.byInvoice["e4b6c2b1-f835-42b7-a06c-fd2f1a455f55"] = "payment-id"

	processPayment := NewProcessPayment(
		repository,
		&invoicePaymentPortFake{},
		gatewayFake{result: paymentports.GatewayResult{Status: "Approved", GatewayReference: "mock-approved"}},
		txManagerFake{},
	)

	_, err := processPayment.Execute(context.Background(), ProcessPaymentInput{
		InvoiceID: "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
	})
	if !errors.Is(err, entities.ErrPaymentAlreadyExists) {
		t.Fatalf("expected duplicate payment error, got %v", err)
	}
}

func TestProcessPaymentPersistsFailureWithoutMarkingInvoicePaid(t *testing.T) {
	t.Parallel()

	repository := newPaymentRepositoryFake()
	invoicePort := &invoicePaymentPortFake{
		snapshot: invoiceports.InvoiceSnapshot{
			ID:          "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
			CustomerID:  "1d7349c1-0d5d-4fd6-9cfe-dcb12e6aa9f5",
			AmountCents: 4500,
			Status:      "Pending",
			DueDate:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		},
	}
	processPayment := NewProcessPayment(
		repository,
		invoicePort,
		gatewayFake{result: paymentports.GatewayResult{Status: "Failed", GatewayReference: "mock-failed"}},
		txManagerFake{},
	)
	processPayment.now = func() time.Time { return time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC) }

	payment, err := processPayment.Execute(context.Background(), ProcessPaymentInput{
		InvoiceID: "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
	})
	if err != nil {
		t.Fatalf("process failed payment: %v", err)
	}

	if payment.Status != "Failed" {
		t.Fatalf("expected failed payment, got %q", payment.Status)
	}

	if invoicePort.markedAsPaid {
		t.Fatal("did not expect invoice to be marked as paid")
	}

	if len(repository.byID) != 1 {
		t.Fatalf("expected failed payment to be persisted, got %d records", len(repository.byID))
	}
}

func TestProcessPaymentPropagatesInvoiceErrors(t *testing.T) {
	t.Parallel()

	processPayment := NewProcessPayment(
		newPaymentRepositoryFake(),
		&invoicePaymentPortFake{getErr: invoiceentities.ErrInvoiceNotFound},
		gatewayFake{result: paymentports.GatewayResult{Status: "Approved", GatewayReference: "mock-approved"}},
		txManagerFake{},
	)

	_, err := processPayment.Execute(context.Background(), ProcessPaymentInput{
		InvoiceID: "e4b6c2b1-f835-42b7-a06c-fd2f1a455f55",
	})
	if !errors.Is(err, invoiceentities.ErrInvoiceNotFound) {
		t.Fatalf("expected invoice not found error, got %v", err)
	}
}
