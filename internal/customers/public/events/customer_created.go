package events

import (
	"context"
	"time"

	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

const EventNameCustomerCreated = "CustomerCreated"

type CustomerCreatedPayload struct {
	CustomerID string `json:"customer_id"`
}

type CustomerCreated struct {
	sharedevent.Envelope[CustomerCreatedPayload]
}

func NewCustomerCreated(ctx context.Context, customerID string, occurredAt time.Time) CustomerCreated {
	return CustomerCreated{
		Envelope: sharedevent.NewEnvelope(
			ctx,
			EventNameCustomerCreated,
			customerID,
			occurredAt,
			CustomerCreatedPayload{
				CustomerID: customerID,
			},
		),
	}
}

func (CustomerCreated) Name() string {
	return EventNameCustomerCreated
}
