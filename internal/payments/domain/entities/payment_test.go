package entities

import (
	"testing"
	"time"
)

func TestPaymentTransitions(t *testing.T) {
	t.Parallel()

	payment, err := NewPayment("payment-id", "invoice-id", time.Now())
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}

	if payment.Status() != StatusPending {
		t.Fatalf("expected pending payment, got %q", payment.Status())
	}

	payment.MarkApproved("mock-ref", time.Now().Add(time.Minute))
	if payment.Status() != StatusApproved {
		t.Fatalf("expected approved payment, got %q", payment.Status())
	}

	payment.MarkFailed("mock-fail", time.Now().Add(2*time.Minute))
	if payment.Status() != StatusFailed {
		t.Fatalf("expected failed payment, got %q", payment.Status())
	}
}

func TestNewPaymentRequiresInvoiceReference(t *testing.T) {
	t.Parallel()

	if _, err := NewPayment("payment-id", "", time.Now()); err != ErrInvalidInvoiceReference {
		t.Fatalf("expected invalid invoice reference, got %v", err)
	}
}
