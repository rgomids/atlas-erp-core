package entities

import (
	"testing"
	"time"
)

func TestPaymentTransitions(t *testing.T) {
	t.Parallel()

	payment, err := NewPayment("payment-id", "billing-id", "invoice-id", 1, "billing:billing-id:attempt:1", time.Now())
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

	payment.MarkFailed("mock-fail", FailureCategoryGatewayDeclined, time.Now().Add(2*time.Minute))
	if payment.Status() != StatusFailed {
		t.Fatalf("expected failed payment, got %q", payment.Status())
	}

	if payment.FailureCategory() != FailureCategoryGatewayDeclined {
		t.Fatalf("expected gateway_declined failure category, got %q", payment.FailureCategory())
	}
}

func TestNewPaymentCreatesPendingPayment(t *testing.T) {
	t.Parallel()

	payment, err := NewPayment("payment-id", "billing-id", "invoice-id", 1, "billing:billing-id:attempt:1", time.Now())
	if err != nil {
		t.Fatalf("expected payment to be created, got %v", err)
	}

	if payment.Status() != StatusPending {
		t.Fatalf("expected pending payment, got %q", payment.Status())
	}

	if payment.InvoiceID() != "invoice-id" {
		t.Fatalf("expected invoice reference to be preserved, got %q", payment.InvoiceID())
	}

	if payment.AttemptNumber() != 1 {
		t.Fatalf("expected attempt number to be preserved, got %d", payment.AttemptNumber())
	}
}

func TestNewPaymentRequiresInvoiceReference(t *testing.T) {
	t.Parallel()

	if _, err := NewPayment("payment-id", "billing-id", "", 1, "billing:billing-id:attempt:1", time.Now()); err != ErrInvalidInvoiceReference {
		t.Fatalf("expected invalid invoice reference, got %v", err)
	}
}

func TestNewPaymentRequiresIdempotencyKey(t *testing.T) {
	t.Parallel()

	if _, err := NewPayment("payment-id", "billing-id", "invoice-id", 1, "", time.Now()); err != ErrInvalidIdempotencyKey {
		t.Fatalf("expected invalid idempotency key, got %v", err)
	}
}
