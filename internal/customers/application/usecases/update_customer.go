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

type UpdateCustomerInput struct {
	ID    string
	Name  string
	Email string
}

type UpdateCustomer struct {
	repository repositories.CustomerRepository
	now        func() time.Time
}

func NewUpdateCustomer(repository repositories.CustomerRepository) UpdateCustomer {
	return UpdateCustomer{
		repository: repository,
		now:        time.Now,
	}
}

func (usecase UpdateCustomer) Execute(ctx context.Context, input UpdateCustomerInput) (dto.Customer, error) {
	customerID, err := uuid.Parse(input.ID)
	if err != nil {
		return dto.Customer{}, entities.ErrInvalidCustomerID
	}

	customer, err := usecase.repository.GetByID(ctx, customerID.String())
	if err != nil {
		return dto.Customer{}, fmt.Errorf("get customer: %w", err)
	}

	if err := customer.UpdateProfile(input.Name, input.Email, usecase.now()); err != nil {
		return dto.Customer{}, err
	}

	if err := usecase.repository.Update(ctx, customer); err != nil {
		return dto.Customer{}, fmt.Errorf("update customer: %w", err)
	}

	return mappers.ToCustomerDTO(customer), nil
}
