package events

import (
	"context"
	"time"

	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

const EventNamePaymentApproved = "PaymentApproved"

type PaymentApprovedPayload struct {
	PaymentID        string `json:"payment_id"`
	BillingID        string `json:"billing_id"`
	InvoiceID        string `json:"invoice_id"`
	CustomerID       string `json:"customer_id"`
	AttemptNumber    int    `json:"attempt_number"`
	IdempotencyKey   string `json:"idempotency_key"`
	GatewayReference string `json:"gateway_reference"`
}

type PaymentApproved struct {
	sharedevent.Envelope[PaymentApprovedPayload]
}

func NewPaymentApproved(
	ctx context.Context,
	paymentID string,
	billingID string,
	invoiceID string,
	customerID string,
	attemptNumber int,
	idempotencyKey string,
	gatewayReference string,
	occurredAt time.Time,
) PaymentApproved {
	return PaymentApproved{
		Envelope: sharedevent.NewEnvelope(
			ctx,
			EventNamePaymentApproved,
			paymentID,
			occurredAt,
			PaymentApprovedPayload{
				PaymentID:        paymentID,
				BillingID:        billingID,
				InvoiceID:        invoiceID,
				CustomerID:       customerID,
				AttemptNumber:    attemptNumber,
				IdempotencyKey:   idempotencyKey,
				GatewayReference: gatewayReference,
			},
		),
	}
}

func (PaymentApproved) Name() string {
	return EventNamePaymentApproved
}
