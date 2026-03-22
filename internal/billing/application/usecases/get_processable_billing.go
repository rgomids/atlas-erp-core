package usecases

import (
	"context"
	"time"

	"github.com/rgomids/atlas-erp-core/internal/billing/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/billing/domain/repositories"
)

type GetProcessableBillingByInvoiceID struct {
	repository repositories.BillingRepository
	now        func() time.Time
}

func NewGetProcessableBillingByInvoiceID(repository repositories.BillingRepository) GetProcessableBillingByInvoiceID {
	return GetProcessableBillingByInvoiceID{
		repository: repository,
		now:        time.Now,
	}
}

func (usecase GetProcessableBillingByInvoiceID) Execute(ctx context.Context, invoiceID string) (ports.BillingSnapshot, error) {
	billing, err := usecase.repository.GetByInvoiceID(ctx, invoiceID)
	if err != nil {
		return ports.BillingSnapshot{}, err
	}

	if err := billing.MarkRequested(usecase.now()); err != nil {
		return ports.BillingSnapshot{}, err
	}

	if err := usecase.repository.Update(ctx, billing); err != nil {
		return ports.BillingSnapshot{}, err
	}

	return toSnapshot(billing), nil
}
