package events

import (
	"context"
	"time"

	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

const EventNameBillingRequested = "BillingRequested"

type BillingRequestedPayload struct {
	BillingID     string    `json:"billing_id"`
	InvoiceID     string    `json:"invoice_id"`
	CustomerID    string    `json:"customer_id"`
	AmountCents   int64     `json:"amount_cents"`
	DueDate       time.Time `json:"due_date"`
	AttemptNumber int       `json:"attempt_number"`
}

type BillingRequested struct {
	sharedevent.Envelope[BillingRequestedPayload]
}

func NewBillingRequested(
	ctx context.Context,
	billingID string,
	invoiceID string,
	customerID string,
	amountCents int64,
	dueDate time.Time,
	attemptNumber int,
	occurredAt time.Time,
) BillingRequested {
	return BillingRequested{
		Envelope: sharedevent.NewEnvelope(
			ctx,
			EventNameBillingRequested,
			billingID,
			occurredAt,
			BillingRequestedPayload{
				BillingID:     billingID,
				InvoiceID:     invoiceID,
				CustomerID:    customerID,
				AmountCents:   amountCents,
				DueDate:       dueDate,
				AttemptNumber: attemptNumber,
			},
		),
	}
}

func (BillingRequested) Name() string {
	return EventNameBillingRequested
}
