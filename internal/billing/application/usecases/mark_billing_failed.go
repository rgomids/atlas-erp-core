package usecases

import (
	"context"
	"time"

	"github.com/rgomids/atlas-erp-core/internal/billing/domain/repositories"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
	"go.opentelemetry.io/otel/attribute"
)

type MarkBillingFailed struct {
	repository    repositories.BillingRepository
	now           func() time.Time
	observability *observability.Runtime
}

func NewMarkBillingFailed(repository repositories.BillingRepository, telemetry ...*observability.Runtime) MarkBillingFailed {
	return MarkBillingFailed{
		repository:    repository,
		now:           time.Now,
		observability: observability.FromOptional(telemetry...),
	}
}

func (usecase MarkBillingFailed) Execute(ctx context.Context, billingID string) (err error) {
	errorType := ""
	ctx, span := usecase.observability.StartUseCase(
		ctx,
		"billing",
		"MarkBillingFailed",
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

	billing.MarkFailed(usecase.now())
	if err := usecase.repository.Update(ctx, billing); err != nil {
		errorType = observability.ErrorTypeInfrastructure
		return err
	}

	return nil
}
