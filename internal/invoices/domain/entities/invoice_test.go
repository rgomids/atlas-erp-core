package entities

import (
	"errors"
	"testing"
	"time"
)

func TestNewInvoiceRequiresPositiveAmountAndDueDate(t *testing.T) {
	t.Parallel()

	_, err := NewInvoice("invoice-id", "customer-id", 0, time.Now(), time.Now())
	if !errors.Is(err, ErrInvoiceAmountMustBePositive) {
		t.Fatalf("expected invalid amount error, got %v", err)
	}

	_, err = NewInvoice("invoice-id", "customer-id", 100, time.Time{}, time.Now())
	if !errors.Is(err, ErrInvoiceDueDateRequired) {
		t.Fatalf("expected due date required error, got %v", err)
	}
}

func TestInvoiceMarkPaidMakesInvoiceImmutable(t *testing.T) {
	t.Parallel()

	invoice, err := NewInvoice("invoice-id", "customer-id", 1000, time.Now().Add(24*time.Hour), time.Now())
	if err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	if err := invoice.MarkPaid(time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("mark paid: %v", err)
	}

	if invoice.Status() != StatusPaid {
		t.Fatalf("expected paid invoice, got %q", invoice.Status())
	}

	if err := invoice.MarkPaid(time.Now().Add(2 * time.Hour)); !errors.Is(err, ErrInvoiceImmutable) {
		t.Fatalf("expected immutable invoice error, got %v", err)
	}
}
