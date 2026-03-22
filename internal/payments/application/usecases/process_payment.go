package usecases

import (
	"context"

	"github.com/google/uuid"

	billingports "github.com/rgomids/atlas-erp-core/internal/billing/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/payments/application/dto"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
)

type ProcessPaymentInput struct {
	InvoiceID string
}

type ProcessPayment struct {
	billingPort           billingports.PaymentCompatibilityPort
	processBillingRequest ProcessBillingRequest
}

func NewProcessPayment(
	billingPort billingports.PaymentCompatibilityPort,
	processBillingRequest ProcessBillingRequest,
) ProcessPayment {
	return ProcessPayment{
		billingPort:           billingPort,
		processBillingRequest: processBillingRequest,
	}
}

func (usecase ProcessPayment) Execute(ctx context.Context, input ProcessPaymentInput) (dto.Payment, error) {
	invoiceID, err := uuid.Parse(input.InvoiceID)
	if err != nil {
		return dto.Payment{}, entities.ErrInvalidInvoiceReference
	}

	billing, err := usecase.billingPort.GetProcessableBillingByInvoiceID(ctx, invoiceID.String())
	if err != nil {
		return dto.Payment{}, err
	}

	return usecase.processBillingRequest.Execute(ctx, ProcessBillingRequestInput{
		BillingID:   billing.ID,
		InvoiceID:   billing.InvoiceID,
		AmountCents: billing.AmountCents,
		DueDate:     billing.DueDate,
	})
}
