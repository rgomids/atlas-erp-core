package usecases

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/rgomids/atlas-erp-core/internal/invoices/application/dto"
	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/repositories"
	"github.com/rgomids/atlas-erp-core/internal/invoices/infrastructure/mappers"
)

type ListCustomerInvoicesInput struct {
	CustomerID string
}

type ListCustomerInvoices struct {
	repository repositories.InvoiceRepository
}

func NewListCustomerInvoices(repository repositories.InvoiceRepository) ListCustomerInvoices {
	return ListCustomerInvoices{repository: repository}
}

func (usecase ListCustomerInvoices) Execute(ctx context.Context, input ListCustomerInvoicesInput) ([]dto.Invoice, error) {
	customerID, err := uuid.Parse(input.CustomerID)
	if err != nil {
		return nil, entities.ErrInvalidCustomerReference
	}

	invoices, err := usecase.repository.ListByCustomerID(ctx, customerID.String())
	if err != nil {
		return nil, fmt.Errorf("list customer invoices: %w", err)
	}

	return mappers.ToInvoiceDTOs(invoices), nil
}
