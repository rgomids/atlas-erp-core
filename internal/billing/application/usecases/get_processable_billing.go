package usecases

import (
	"context"
	"time"

	"github.com/rgomids/atlas-erp-core/internal/billing/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/billing/domain/repositories"
)

type GetProcessableBillingByInvoiceID struct {
	repository         repositories.BillingRepository
	transactionManager ports.TransactionManager
	now                func() time.Time
}

func NewGetProcessableBillingByInvoiceID(
	repository repositories.BillingRepository,
	transactionManager ports.TransactionManager,
) GetProcessableBillingByInvoiceID {
	return GetProcessableBillingByInvoiceID{
		repository:         repository,
		transactionManager: transactionManager,
		now:                time.Now,
	}
}

func (usecase GetProcessableBillingByInvoiceID) Execute(ctx context.Context, invoiceID string) (ports.BillingSnapshot, error) {
	var snapshot ports.BillingSnapshot
	err := usecase.transactionManager.WithinTransaction(ctx, func(txContext context.Context) error {
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
		return ports.BillingSnapshot{}, err
	}

	return snapshot, nil
}
