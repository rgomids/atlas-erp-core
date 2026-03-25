package events

import (
	"context"
	"time"

	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

const EventNameInvoicePaid = "InvoicePaid"

type InvoicePaidPayload struct {
	InvoiceID string `json:"invoice_id"`
}

type InvoicePaid struct {
	sharedevent.Envelope[InvoicePaidPayload]
}

func NewInvoicePaid(ctx context.Context, invoiceID string, occurredAt time.Time) InvoicePaid {
	return InvoicePaid{
		Envelope: sharedevent.NewEnvelope(
			ctx,
			EventNameInvoicePaid,
			invoiceID,
			occurredAt,
			InvoicePaidPayload{
				InvoiceID: invoiceID,
			},
		),
	}
}

func (InvoicePaid) Name() string {
	return EventNameInvoicePaid
}
