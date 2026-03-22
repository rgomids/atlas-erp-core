package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	customerports "github.com/rgomids/atlas-erp-core/internal/customers/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/invoices/application/dto"
	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/entities"
	invoiceevents "github.com/rgomids/atlas-erp-core/internal/invoices/domain/events"
	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/repositories"
	"github.com/rgomids/atlas-erp-core/internal/invoices/infrastructure/mappers"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
)

type CreateInvoiceInput struct {
	CustomerID  string
	AmountCents int64
	DueDate     string
}

type CreateInvoice struct {
	repository             repositories.InvoiceRepository
	customerExistenceCheck customerports.ExistenceChecker
	bus                    sharedevent.EventBus
	now                    func() time.Time
	observability          *observability.Runtime
}

func NewCreateInvoice(
	repository repositories.InvoiceRepository,
	customerExistenceCheck customerports.ExistenceChecker,
	bus sharedevent.EventBus,
	telemetry ...*observability.Runtime,
) CreateInvoice {
	return CreateInvoice{
		repository:             repository,
		customerExistenceCheck: customerExistenceCheck,
		bus:                    bus,
		now:                    time.Now,
		observability:          observability.FromOptional(telemetry...),
	}
}

func (usecase CreateInvoice) Execute(ctx context.Context, input CreateInvoiceInput) (invoiceDTO dto.Invoice, err error) {
	errorType := ""
	ctx, span := usecase.observability.StartUseCase(ctx, "invoices", "CreateInvoice")
	defer func() {
		usecase.observability.CompleteSpan(span, err, errorType)
	}()

	customerID, err := uuid.Parse(input.CustomerID)
	if err != nil {
		errorType = observability.ErrorTypeValidation
		return dto.Invoice{}, entities.ErrInvalidCustomerReference
	}

	if err := usecase.customerExistenceCheck.ExistsActiveCustomer(ctx, customerID.String()); err != nil {
		errorType = observability.ErrorTypeDomain
		return dto.Invoice{}, err
	}

	dueDate, err := time.Parse("2006-01-02", input.DueDate)
	if err != nil {
		errorType = observability.ErrorTypeValidation
		return dto.Invoice{}, entities.ErrInvoiceDueDateRequired
	}

	invoice, err := entities.NewInvoice(uuid.NewString(), customerID.String(), input.AmountCents, dueDate, usecase.now())
	if err != nil {
		errorType = observability.ErrorTypeValidation
		return dto.Invoice{}, err
	}

	if err := usecase.repository.Save(ctx, invoice); err != nil {
		errorType = observability.ErrorTypeInfrastructure
		return dto.Invoice{}, fmt.Errorf("save invoice: %w", err)
	}

	span.SetAttributes(
		attribute.String("atlas.customer_id", invoice.CustomerID()),
		attribute.String("atlas.invoice_id", invoice.ID()),
	)

	if err := sharedevent.Publish(ctx, usecase.bus, "invoices", invoiceevents.InvoiceCreated{
		InvoiceID:   invoice.ID(),
		CustomerID:  invoice.CustomerID(),
		AmountCents: invoice.AmountCents(),
		DueDate:     invoice.DueDate(),
		CreatedAt:   invoice.CreatedAt(),
	}); err != nil {
		errorType = observability.ErrorTypeInfrastructure
		return dto.Invoice{}, err
	}

	return mappers.ToInvoiceDTO(invoice), nil
}
