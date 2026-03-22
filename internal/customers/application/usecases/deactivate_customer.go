package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/rgomids/atlas-erp-core/internal/customers/application/dto"
	"github.com/rgomids/atlas-erp-core/internal/customers/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/customers/domain/repositories"
	"github.com/rgomids/atlas-erp-core/internal/customers/infrastructure/mappers"
)

type DeactivateCustomerInput struct {
	ID string
}

type DeactivateCustomer struct {
	repository repositories.CustomerRepository
	now        func() time.Time
}

func NewDeactivateCustomer(repository repositories.CustomerRepository) DeactivateCustomer {
	return DeactivateCustomer{
		repository: repository,
		now:        time.Now,
	}
}

func (usecase DeactivateCustomer) Execute(ctx context.Context, input DeactivateCustomerInput) (dto.Customer, error) {
	customerID, err := uuid.Parse(input.ID)
	if err != nil {
		return dto.Customer{}, entities.ErrInvalidCustomerID
	}

	customer, err := usecase.repository.GetByID(ctx, customerID.String())
	if err != nil {
		return dto.Customer{}, fmt.Errorf("get customer: %w", err)
	}

	customer.Deactivate(usecase.now())

	if err := usecase.repository.Update(ctx, customer); err != nil {
		return dto.Customer{}, fmt.Errorf("deactivate customer: %w", err)
	}

	return mappers.ToCustomerDTO(customer), nil
}
