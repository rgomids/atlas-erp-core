package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/rgomids/atlas-erp-core/internal/customers/application/dto"
	"github.com/rgomids/atlas-erp-core/internal/customers/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/customers/domain/repositories"
	"github.com/rgomids/atlas-erp-core/internal/customers/infrastructure/mappers"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
)

type DeactivateCustomerInput struct {
	ID string
}

type DeactivateCustomer struct {
	repository    repositories.CustomerRepository
	now           func() time.Time
	observability *observability.Runtime
}

func NewDeactivateCustomer(repository repositories.CustomerRepository, telemetry ...*observability.Runtime) DeactivateCustomer {
	return DeactivateCustomer{
		repository:    repository,
		now:           time.Now,
		observability: observability.FromOptional(telemetry...),
	}
}

func (usecase DeactivateCustomer) Execute(ctx context.Context, input DeactivateCustomerInput) (customerDTO dto.Customer, err error) {
	errorType := ""
	ctx, span := usecase.observability.StartUseCase(ctx, "customers", "DeactivateCustomer")
	defer func() {
		usecase.observability.CompleteSpan(span, err, errorType)
	}()

	customerID, err := uuid.Parse(input.ID)
	if err != nil {
		errorType = observability.ErrorTypeValidation
		return dto.Customer{}, entities.ErrInvalidCustomerID
	}

	customer, err := usecase.repository.GetByID(ctx, customerID.String())
	if err != nil {
		errorType = observability.ErrorTypeInfrastructure
		return dto.Customer{}, fmt.Errorf("get customer: %w", err)
	}

	customer.Deactivate(usecase.now())

	if err := usecase.repository.Update(ctx, customer); err != nil {
		errorType = observability.ErrorTypeInfrastructure
		return dto.Customer{}, fmt.Errorf("deactivate customer: %w", err)
	}

	span.SetAttributes(attribute.String("atlas.customer_id", customer.ID()))

	return mappers.ToCustomerDTO(customer), nil
}
