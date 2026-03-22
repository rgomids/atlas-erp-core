package handlers

import (
	"context"
	"fmt"

	"github.com/rgomids/atlas-erp-core/internal/billing/application/usecases"
	paymentevents "github.com/rgomids/atlas-erp-core/internal/payments/domain/events"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

type PaymentFailed struct {
	markBillingFailed usecases.MarkBillingFailed
}

func NewPaymentFailed(markBillingFailed usecases.MarkBillingFailed) PaymentFailed {
	return PaymentFailed{markBillingFailed: markBillingFailed}
}

func (handler PaymentFailed) Handle(ctx context.Context, event sharedevent.Event) error {
	domainEvent, ok := event.(paymentevents.PaymentFailed)
	if !ok {
		return fmt.Errorf("unexpected event type %T", event)
	}

	return handler.markBillingFailed.Execute(ctx, domainEvent.BillingID)
}
