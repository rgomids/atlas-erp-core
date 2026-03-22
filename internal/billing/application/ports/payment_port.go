package ports

import (
	"context"
	"time"
)

type BillingSnapshot struct {
	ID          string
	InvoiceID   string
	AmountCents int64
	DueDate     time.Time
	Status      string
}

type PaymentCompatibilityPort interface {
	GetProcessableBillingByInvoiceID(ctx context.Context, invoiceID string) (BillingSnapshot, error)
}
