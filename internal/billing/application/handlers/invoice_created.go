package handlers

import (
	"context"
	"fmt"

	"github.com/rgomids/atlas-erp-core/internal/billing/application/usecases"
	invoiceevents "github.com/rgomids/atlas-erp-core/internal/invoices/domain/events"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

type InvoiceCreated struct {
	createBilling usecases.CreateBillingFromInvoice
}

func NewInvoiceCreated(createBilling usecases.CreateBillingFromInvoice) InvoiceCreated {
	return InvoiceCreated{createBilling: createBilling}
}

func (handler InvoiceCreated) Handle(ctx context.Context, event sharedevent.Event) error {
	domainEvent, ok := event.(invoiceevents.InvoiceCreated)
	if !ok {
		return fmt.Errorf("unexpected event type %T", event)
	}

	_, err := handler.createBilling.Execute(ctx, usecases.CreateBillingFromInvoiceInput{
		InvoiceID:   domainEvent.InvoiceID,
		CustomerID:  domainEvent.CustomerID,
		AmountCents: domainEvent.AmountCents,
		DueDate:     domainEvent.DueDate,
	})
	return err
}
