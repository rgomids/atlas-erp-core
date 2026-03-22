package usecases

import (
	"context"
	"errors"

	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/entities"
	invoiceevents "github.com/rgomids/atlas-erp-core/internal/invoices/domain/events"
	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/repositories"
	paymentevents "github.com/rgomids/atlas-erp-core/internal/payments/domain/events"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

type ApplyPaymentApproved struct {
	repository repositories.InvoiceRepository
	bus        sharedevent.EventBus
}

func NewApplyPaymentApproved(repository repositories.InvoiceRepository, bus sharedevent.EventBus) ApplyPaymentApproved {
	return ApplyPaymentApproved{
		repository: repository,
		bus:        bus,
	}
}

func (usecase ApplyPaymentApproved) Execute(ctx context.Context, paymentApproved paymentevents.PaymentApproved) error {
	invoice, err := usecase.repository.GetByID(ctx, paymentApproved.InvoiceID)
	if err != nil {
		return err
	}

	if err := invoice.MarkPaid(paymentApproved.ApprovedAt); err != nil {
		if errors.Is(err, entities.ErrInvoiceImmutable) {
			return nil
		}

		return err
	}

	if err := usecase.repository.Update(ctx, invoice); err != nil {
		return err
	}

	return sharedevent.Publish(ctx, usecase.bus, "invoices", invoiceevents.InvoicePaid{
		InvoiceID: invoice.ID(),
		PaidAt:    paymentApproved.ApprovedAt,
	})
}
