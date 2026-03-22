package usecases

import (
	"context"
	"time"

	"github.com/rgomids/atlas-erp-core/internal/billing/domain/repositories"
)

type MarkBillingFailed struct {
	repository repositories.BillingRepository
	now        func() time.Time
}

func NewMarkBillingFailed(repository repositories.BillingRepository) MarkBillingFailed {
	return MarkBillingFailed{
		repository: repository,
		now:        time.Now,
	}
}

func (usecase MarkBillingFailed) Execute(ctx context.Context, billingID string) error {
	billing, err := usecase.repository.GetByID(ctx, billingID)
	if err != nil {
		return err
	}

	billing.MarkFailed(usecase.now())
	return usecase.repository.Update(ctx, billing)
}
