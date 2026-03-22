package handlers

import (
	"context"
	"fmt"

	"github.com/rgomids/atlas-erp-core/internal/invoices/application/usecases"
	paymentevents "github.com/rgomids/atlas-erp-core/internal/payments/domain/events"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

type PaymentApproved struct {
	applyPaymentApproved usecases.ApplyPaymentApproved
}

func NewPaymentApproved(applyPaymentApproved usecases.ApplyPaymentApproved) PaymentApproved {
	return PaymentApproved{applyPaymentApproved: applyPaymentApproved}
}

func (handler PaymentApproved) Handle(ctx context.Context, event sharedevent.Event) error {
	domainEvent, ok := event.(paymentevents.PaymentApproved)
	if !ok {
		return fmt.Errorf("unexpected event type %T", event)
	}

	return handler.applyPaymentApproved.Execute(ctx, domainEvent)
}
