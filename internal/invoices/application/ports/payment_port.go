package ports

import (
	"context"
	"time"
)

type InvoiceSnapshot struct {
	ID          string
	CustomerID  string
	AmountCents int64
	DueDate     time.Time
	Status      string
}

type InvoicePaymentPort interface {
	GetPayableInvoice(ctx context.Context, invoiceID string) (InvoiceSnapshot, error)
	MarkAsPaid(ctx context.Context, invoiceID string, paidAt time.Time) error
}
