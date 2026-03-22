package usecases

import (
	"context"
	"time"

	"github.com/rgomids/atlas-erp-core/internal/billing/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/billing/domain/repositories"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
	"go.opentelemetry.io/otel/attribute"
)

type GetProcessableBillingByInvoiceID struct {
	repository         repositories.BillingRepository
	transactionManager ports.TransactionManager
	now                func() time.Time
	observability      *observability.Runtime
}

func NewGetProcessableBillingByInvoiceID(
	repository repositories.BillingRepository,
	transactionManager ports.TransactionManager,
	telemetry ...*observability.Runtime,
) GetProcessableBillingByInvoiceID {
	return GetProcessableBillingByInvoiceID{
		repository:         repository,
		transactionManager: transactionManager,
		now:                time.Now,
		observability:      observability.FromOptional(telemetry...),
	}
}

func (usecase GetProcessableBillingByInvoiceID) Execute(ctx context.Context, invoiceID string) (snapshot ports.BillingSnapshot, err error) {
	errorType := ""
	ctx, span := usecase.observability.StartUseCase(
		ctx,
		"billing",
		"GetProcessableBillingByInvoiceID",
		attribute.String("atlas.invoice_id", invoiceID),
	)
	defer func() {
		usecase.observability.CompleteSpan(span, err, errorType)
	}()

	err = usecase.transactionManager.WithinTransaction(ctx, func(txContext context.Context) error {
		billing, err := usecase.repository.GetByInvoiceIDForUpdate(txContext, invoiceID)
		if err != nil {
			return err
		}

		if err := billing.MarkRequested(usecase.now()); err != nil {
			return err
		}

		if err := usecase.repository.Update(txContext, billing); err != nil {
			return err
		}

		snapshot = toSnapshot(billing)
		return nil
	})
	if err != nil {
		errorType = observability.ErrorTypeInfrastructure
		return ports.BillingSnapshot{}, err
	}

	span.SetAttributes(
		attribute.String("atlas.billing_id", snapshot.ID),
		attribute.String("atlas.customer_id", snapshot.CustomerID),
	)

	return snapshot, nil
}
