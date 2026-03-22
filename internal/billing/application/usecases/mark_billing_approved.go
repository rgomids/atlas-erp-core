package usecases

import (
	"context"
	"time"

	"github.com/rgomids/atlas-erp-core/internal/billing/domain/repositories"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
	"go.opentelemetry.io/otel/attribute"
)

type MarkBillingApproved struct {
	repository    repositories.BillingRepository
	now           func() time.Time
	observability *observability.Runtime
}

func NewMarkBillingApproved(repository repositories.BillingRepository, telemetry ...*observability.Runtime) MarkBillingApproved {
	return MarkBillingApproved{
		repository:    repository,
		now:           time.Now,
		observability: observability.FromOptional(telemetry...),
	}
}

func (usecase MarkBillingApproved) Execute(ctx context.Context, billingID string) (err error) {
	errorType := ""
	ctx, span := usecase.observability.StartUseCase(
		ctx,
		"billing",
		"MarkBillingApproved",
		attribute.String("atlas.billing_id", billingID),
	)
	defer func() {
		usecase.observability.CompleteSpan(span, err, errorType)
	}()

	billing, err := usecase.repository.GetByID(ctx, billingID)
	if err != nil {
		errorType = observability.ErrorTypeInfrastructure
		return err
	}

	billing.MarkApproved(usecase.now())
	if err := usecase.repository.Update(ctx, billing); err != nil {
		errorType = observability.ErrorTypeInfrastructure
		return err
	}

	return nil
}
