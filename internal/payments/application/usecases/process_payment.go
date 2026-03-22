package usecases

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	billingports "github.com/rgomids/atlas-erp-core/internal/billing/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/payments/application/dto"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
)

type ProcessPaymentInput struct {
	InvoiceID string
}

type ProcessPayment struct {
	billingPort           billingports.PaymentCompatibilityPort
	processBillingRequest ProcessBillingRequest
	observability         *observability.Runtime
}

func NewProcessPayment(
	billingPort billingports.PaymentCompatibilityPort,
	processBillingRequest ProcessBillingRequest,
	telemetry ...*observability.Runtime,
) ProcessPayment {
	return ProcessPayment{
		billingPort:           billingPort,
		processBillingRequest: processBillingRequest,
		observability:         observability.FromOptional(telemetry...),
	}
}

func (usecase ProcessPayment) Execute(ctx context.Context, input ProcessPaymentInput) (payment dto.Payment, err error) {
	errorType := ""
	ctx, span := usecase.observability.StartUseCase(
		ctx,
		"payments",
		"ProcessPayment",
		attribute.String("atlas.invoice_id", input.InvoiceID),
	)
	defer func() {
		usecase.observability.CompleteSpan(span, err, errorType)
	}()

	invoiceID, err := uuid.Parse(input.InvoiceID)
	if err != nil {
		errorType = observability.ErrorTypeValidation
		return dto.Payment{}, entities.ErrInvalidInvoiceReference
	}

	billing, err := usecase.billingPort.GetProcessableBillingByInvoiceID(ctx, invoiceID.String())
	if err != nil {
		errorType = observability.ErrorTypeDomain
		return dto.Payment{}, err
	}

	span.SetAttributes(
		attribute.String("atlas.billing_id", billing.ID),
		attribute.String("atlas.customer_id", billing.CustomerID),
	)

	return usecase.processBillingRequest.Execute(ctx, ProcessBillingRequestInput{
		BillingID:     billing.ID,
		InvoiceID:     billing.InvoiceID,
		CustomerID:    billing.CustomerID,
		AmountCents:   billing.AmountCents,
		DueDate:       billing.DueDate,
		AttemptNumber: billing.AttemptNumber,
	})
}
