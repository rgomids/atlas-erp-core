package events

import (
	"context"
	"time"

	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

const EventNameInvoiceCreated = "InvoiceCreated"

type InvoiceCreatedPayload struct {
	InvoiceID   string    `json:"invoice_id"`
	CustomerID  string    `json:"customer_id"`
	AmountCents int64     `json:"amount_cents"`
	DueDate     time.Time `json:"due_date"`
}

type InvoiceCreated struct {
	sharedevent.Envelope[InvoiceCreatedPayload]
}

func NewInvoiceCreated(
	ctx context.Context,
	invoiceID string,
	customerID string,
	amountCents int64,
	dueDate time.Time,
	occurredAt time.Time,
) InvoiceCreated {
	return InvoiceCreated{
		Envelope: sharedevent.NewEnvelope(
			ctx,
			EventNameInvoiceCreated,
			invoiceID,
			occurredAt,
			InvoiceCreatedPayload{
				InvoiceID:   invoiceID,
				CustomerID:  customerID,
				AmountCents: amountCents,
				DueDate:     dueDate,
			},
		),
	}
}

func (InvoiceCreated) Name() string {
	return EventNameInvoiceCreated
}
