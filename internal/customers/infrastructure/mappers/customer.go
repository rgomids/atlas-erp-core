package mappers

import (
	"github.com/rgomids/atlas-erp-core/internal/customers/application/dto"
	"github.com/rgomids/atlas-erp-core/internal/customers/domain/entities"
)

func ToCustomerDTO(customer entities.Customer) dto.Customer {
	return dto.Customer{
		ID:        customer.ID(),
		Name:      customer.Name(),
		Document:  customer.Document().Value(),
		Email:     customer.Email().Value(),
		Status:    string(customer.Status()),
		CreatedAt: customer.CreatedAt(),
		UpdatedAt: customer.UpdatedAt(),
	}
}
