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
	customerevents "github.com/rgomids/atlas-erp-core/internal/customers/public/events"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
)

type CreateCustomerInput struct {
	Name     string
	Document string
	Email    string
}

type CreateCustomer struct {
	repository    repositories.CustomerRepository
	bus           sharedevent.EventBus
	now           func() time.Time
	observability *observability.Runtime
}

func NewCreateCustomer(repository repositories.CustomerRepository, bus sharedevent.EventBus, telemetry ...*observability.Runtime) CreateCustomer {
	return CreateCustomer{
		repository:    repository,
		bus:           bus,
		now:           time.Now,
		observability: observability.FromOptional(telemetry...),
	}
}

func (usecase CreateCustomer) Execute(ctx context.Context, input CreateCustomerInput) (customerDTO dto.Customer, err error) {
	errorType := ""
	ctx, span := usecase.observability.StartUseCase(ctx, "customers", "CreateCustomer")
	defer func() {
		usecase.observability.CompleteSpan(span, err, errorType)
	}()

	customer, err := entities.NewCustomer(uuid.NewString(), input.Name, input.Document, input.Email, usecase.now())
	if err != nil {
		errorType = observability.ErrorTypeValidation
		return dto.Customer{}, err
	}

	exists, err := usecase.repository.ExistsByDocument(ctx, customer.Document().Value())
	if err != nil {
		errorType = observability.ErrorTypeInfrastructure
		return dto.Customer{}, fmt.Errorf("check customer document: %w", err)
	}

	if exists {
		errorType = observability.ErrorTypeDomain
		return dto.Customer{}, entities.ErrCustomerAlreadyExists
	}

	if err := usecase.repository.Save(ctx, customer); err != nil {
		errorType = observability.ErrorTypeInfrastructure
		return dto.Customer{}, fmt.Errorf("save customer: %w", err)
	}

	span.SetAttributes(attribute.String("atlas.customer_id", customer.ID()))

	if err := sharedevent.Publish(
		ctx,
		usecase.bus,
		"customers",
		customerevents.NewCustomerCreated(ctx, customer.ID(), customer.CreatedAt()),
	); err != nil {
		errorType = observability.ErrorTypeInfrastructure
		return dto.Customer{}, err
	}

	return mappers.ToCustomerDTO(customer), nil
}
