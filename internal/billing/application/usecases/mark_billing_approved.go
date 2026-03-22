package usecases

import (
	"context"
	"time"

	"github.com/rgomids/atlas-erp-core/internal/billing/domain/repositories"
)

type MarkBillingApproved struct {
	repository repositories.BillingRepository
	now        func() time.Time
}

func NewMarkBillingApproved(repository repositories.BillingRepository) MarkBillingApproved {
	return MarkBillingApproved{
		repository: repository,
		now:        time.Now,
	}
}

func (usecase MarkBillingApproved) Execute(ctx context.Context, billingID string) error {
	billing, err := usecase.repository.GetByID(ctx, billingID)
	if err != nil {
		return err
	}

	billing.MarkApproved(usecase.now())
	return usecase.repository.Update(ctx, billing)
}
