package public

import (
	"context"
	"time"
)

type BillingSnapshot struct {
	ID            string
	InvoiceID     string
	CustomerID    string
	AmountCents   int64
	DueDate       time.Time
	Status        string
	AttemptNumber int
}

type PaymentCompatibilityPort interface {
	GetProcessableBillingByInvoiceID(ctx context.Context, invoiceID string) (BillingSnapshot, error)
}
