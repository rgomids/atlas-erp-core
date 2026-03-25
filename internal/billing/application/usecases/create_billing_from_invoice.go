package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/rgomids/atlas-erp-core/internal/billing/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/billing/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/billing/domain/repositories"
	billingevents "github.com/rgomids/atlas-erp-core/internal/billing/public/events"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
)

type CreateBillingFromInvoiceInput struct {
	InvoiceID   string
	CustomerID  string
	AmountCents int64
	DueDate     time.Time
}

type CreateBillingFromInvoice struct {
	repository    repositories.BillingRepository
	bus           sharedevent.EventBus
	now           func() time.Time
	observability *observability.Runtime
}

func NewCreateBillingFromInvoice(repository repositories.BillingRepository, bus sharedevent.EventBus, telemetry ...*observability.Runtime) CreateBillingFromInvoice {
	return CreateBillingFromInvoice{
		repository:    repository,
		bus:           bus,
		now:           time.Now,
		observability: observability.FromOptional(telemetry...),
	}
}

func (usecase CreateBillingFromInvoice) Execute(ctx context.Context, input CreateBillingFromInvoiceInput) (snapshot ports.BillingSnapshot, err error) {
	errorType := ""
	ctx, span := usecase.observability.StartUseCase(
		ctx,
		"billing",
		"CreateBillingFromInvoice",
		attribute.String("atlas.invoice_id", input.InvoiceID),
		attribute.String("atlas.customer_id", input.CustomerID),
	)
	defer func() {
		usecase.observability.CompleteSpan(span, err, errorType)
	}()

	if _, err := uuid.Parse(input.InvoiceID); err != nil {
		errorType = observability.ErrorTypeValidation
		return ports.BillingSnapshot{}, entities.ErrInvalidInvoiceReference
	}
	if _, err := uuid.Parse(input.CustomerID); err != nil {
		errorType = observability.ErrorTypeValidation
		return ports.BillingSnapshot{}, entities.ErrInvalidCustomerReference
	}

	existing, err := usecase.repository.GetByInvoiceID(ctx, input.InvoiceID)
	switch {
	case err == nil:
		span.SetAttributes(attribute.String("atlas.billing_id", existing.ID()))
		return toSnapshot(existing), nil
	case !errors.Is(err, entities.ErrBillingNotFound):
		errorType = observability.ErrorTypeInfrastructure
		return ports.BillingSnapshot{}, err
	}

	billing, err := entities.NewBilling(uuid.NewString(), input.InvoiceID, input.CustomerID, input.AmountCents, input.DueDate, usecase.now())
	if err != nil {
		errorType = observability.ErrorTypeDomain
		return ports.BillingSnapshot{}, err
	}

	if err := usecase.repository.Save(ctx, billing); err != nil {
		if errors.Is(err, entities.ErrBillingAlreadyExists) {
			existing, getErr := usecase.repository.GetByInvoiceID(ctx, input.InvoiceID)
			if getErr != nil {
				errorType = observability.ErrorTypeInfrastructure
				return ports.BillingSnapshot{}, getErr
			}

			errorType = observability.ErrorTypeDomain
			return toSnapshot(existing), nil
		}

		errorType = observability.ErrorTypeInfrastructure
		return ports.BillingSnapshot{}, fmt.Errorf("save billing: %w", err)
	}

	span.SetAttributes(attribute.String("atlas.billing_id", billing.ID()))

	if err := sharedevent.Publish(
		ctx,
		usecase.bus,
		"billing",
		billingevents.NewBillingRequested(
			ctx,
			billing.ID(),
			billing.InvoiceID(),
			billing.CustomerID(),
			billing.AmountCents(),
			billing.DueDate(),
			billing.AttemptNumber(),
			billing.UpdatedAt(),
		),
	); err != nil {
		errorType = observability.ErrorTypeInfrastructure
		return ports.BillingSnapshot{}, err
	}

	return toSnapshot(billing), nil
}
