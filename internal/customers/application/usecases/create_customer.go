package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/rgomids/atlas-erp-core/internal/customers/application/dto"
	"github.com/rgomids/atlas-erp-core/internal/customers/domain/entities"
	customerevents "github.com/rgomids/atlas-erp-core/internal/customers/domain/events"
	"github.com/rgomids/atlas-erp-core/internal/customers/domain/repositories"
	"github.com/rgomids/atlas-erp-core/internal/customers/infrastructure/mappers"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

type CreateCustomerInput struct {
	Name     string
	Document string
	Email    string
}

type CreateCustomer struct {
	repository repositories.CustomerRepository
	bus        sharedevent.EventBus
	now        func() time.Time
}

func NewCreateCustomer(repository repositories.CustomerRepository, bus sharedevent.EventBus) CreateCustomer {
	return CreateCustomer{
		repository: repository,
		bus:        bus,
		now:        time.Now,
	}
}

func (usecase CreateCustomer) Execute(ctx context.Context, input CreateCustomerInput) (dto.Customer, error) {
	customer, err := entities.NewCustomer(uuid.NewString(), input.Name, input.Document, input.Email, usecase.now())
	if err != nil {
		return dto.Customer{}, err
	}

	exists, err := usecase.repository.ExistsByDocument(ctx, customer.Document().Value())
	if err != nil {
		return dto.Customer{}, fmt.Errorf("check customer document: %w", err)
	}

	if exists {
		return dto.Customer{}, entities.ErrCustomerAlreadyExists
	}

	if err := usecase.repository.Save(ctx, customer); err != nil {
		return dto.Customer{}, fmt.Errorf("save customer: %w", err)
	}

	if err := sharedevent.Publish(ctx, usecase.bus, "customers", customerevents.CustomerCreated{
		CustomerID: customer.ID(),
	}); err != nil {
		return dto.Customer{}, err
	}

	return mappers.ToCustomerDTO(customer), nil
}
