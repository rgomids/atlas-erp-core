package handlers

import (
	"context"
	"fmt"

	billingevents "github.com/rgomids/atlas-erp-core/internal/billing/domain/events"
	"github.com/rgomids/atlas-erp-core/internal/payments/application/usecases"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

type BillingRequested struct {
	processBillingRequest usecases.ProcessBillingRequest
}

func NewBillingRequested(processBillingRequest usecases.ProcessBillingRequest) BillingRequested {
	return BillingRequested{processBillingRequest: processBillingRequest}
}

func (handler BillingRequested) Handle(ctx context.Context, event sharedevent.Event) error {
	domainEvent, ok := event.(billingevents.BillingRequested)
	if !ok {
		return fmt.Errorf("unexpected event type %T", event)
	}

	_, err := handler.processBillingRequest.Execute(ctx, usecases.ProcessBillingRequestInput{
		BillingID:     domainEvent.BillingID,
		InvoiceID:     domainEvent.InvoiceID,
		CustomerID:    domainEvent.CustomerID,
		AmountCents:   domainEvent.AmountCents,
		DueDate:       domainEvent.DueDate,
		AttemptNumber: domainEvent.AttemptNumber,
	})
	return err
}
