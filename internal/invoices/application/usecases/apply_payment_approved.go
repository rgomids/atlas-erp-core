package usecases

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"

	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/entities"
	invoiceevents "github.com/rgomids/atlas-erp-core/internal/invoices/domain/events"
	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/repositories"
	paymentevents "github.com/rgomids/atlas-erp-core/internal/payments/domain/events"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
)

type ApplyPaymentApproved struct {
	repository    repositories.InvoiceRepository
	bus           sharedevent.EventBus
	observability *observability.Runtime
}

func NewApplyPaymentApproved(repository repositories.InvoiceRepository, bus sharedevent.EventBus, telemetry ...*observability.Runtime) ApplyPaymentApproved {
	return ApplyPaymentApproved{
		repository:    repository,
		bus:           bus,
		observability: observability.FromOptional(telemetry...),
	}
}

func (usecase ApplyPaymentApproved) Execute(ctx context.Context, paymentApproved paymentevents.PaymentApproved) (err error) {
	errorType := ""
	ctx, span := usecase.observability.StartUseCase(
		ctx,
		"invoices",
		"ApplyPaymentApproved",
		attribute.String("atlas.invoice_id", paymentApproved.InvoiceID),
		attribute.String("atlas.payment_id", paymentApproved.PaymentID),
	)
	defer func() {
		usecase.observability.CompleteSpan(span, err, errorType)
	}()

	invoice, err := usecase.repository.GetByID(ctx, paymentApproved.InvoiceID)
	if err != nil {
		errorType = observability.ErrorTypeInfrastructure
		return err
	}

	if err := invoice.MarkPaid(paymentApproved.ApprovedAt); err != nil {
		if errors.Is(err, entities.ErrInvoiceImmutable) {
			errorType = observability.ErrorTypeDomain
			return nil
		}

		errorType = observability.ErrorTypeDomain
		return err
	}

	if err := usecase.repository.Update(ctx, invoice); err != nil {
		errorType = observability.ErrorTypeInfrastructure
		return err
	}

	if err := sharedevent.Publish(ctx, usecase.bus, "invoices", invoiceevents.InvoicePaid{
		InvoiceID: invoice.ID(),
		PaidAt:    paymentApproved.ApprovedAt,
	}); err != nil {
		errorType = observability.ErrorTypeInfrastructure
		return err
	}

	return nil
}
