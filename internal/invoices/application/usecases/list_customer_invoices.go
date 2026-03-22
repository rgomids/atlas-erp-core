package usecases

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/rgomids/atlas-erp-core/internal/invoices/application/dto"
	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/repositories"
	"github.com/rgomids/atlas-erp-core/internal/invoices/infrastructure/mappers"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
)

type ListCustomerInvoicesInput struct {
	CustomerID string
}

type ListCustomerInvoices struct {
	repository    repositories.InvoiceRepository
	observability *observability.Runtime
}

func NewListCustomerInvoices(repository repositories.InvoiceRepository, telemetry ...*observability.Runtime) ListCustomerInvoices {
	return ListCustomerInvoices{
		repository:    repository,
		observability: observability.FromOptional(telemetry...),
	}
}

func (usecase ListCustomerInvoices) Execute(ctx context.Context, input ListCustomerInvoicesInput) (invoiceDTOs []dto.Invoice, err error) {
	errorType := ""
	ctx, span := usecase.observability.StartUseCase(ctx, "invoices", "ListCustomerInvoices")
	defer func() {
		usecase.observability.CompleteSpan(span, err, errorType)
	}()

	customerID, err := uuid.Parse(input.CustomerID)
	if err != nil {
		errorType = observability.ErrorTypeValidation
		return nil, entities.ErrInvalidCustomerReference
	}

	invoices, err := usecase.repository.ListByCustomerID(ctx, customerID.String())
	if err != nil {
		errorType = observability.ErrorTypeInfrastructure
		return nil, fmt.Errorf("list customer invoices: %w", err)
	}

	span.SetAttributes(attribute.String("atlas.customer_id", customerID.String()))

	return mappers.ToInvoiceDTOs(invoices), nil
}
