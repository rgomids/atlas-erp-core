package handlers

import (
	"context"
	"fmt"

	"github.com/rgomids/atlas-erp-core/internal/billing/application/usecases"
	paymentevents "github.com/rgomids/atlas-erp-core/internal/payments/domain/events"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

type PaymentApproved struct {
	markBillingApproved usecases.MarkBillingApproved
}

func NewPaymentApproved(markBillingApproved usecases.MarkBillingApproved) PaymentApproved {
	return PaymentApproved{markBillingApproved: markBillingApproved}
}

func (handler PaymentApproved) Handle(ctx context.Context, event sharedevent.Event) error {
	domainEvent, ok := event.(paymentevents.PaymentApproved)
	if !ok {
		return fmt.Errorf("unexpected event type %T", event)
	}

	return handler.markBillingApproved.Execute(ctx, domainEvent.BillingID)
}
