package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/rgomids/atlas-erp-core/internal/payments/application/dto"
	"github.com/rgomids/atlas-erp-core/internal/payments/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
	paymentevents "github.com/rgomids/atlas-erp-core/internal/payments/domain/events"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/repositories"
	"github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/mappers"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

type ProcessBillingRequestInput struct {
	BillingID   string
	InvoiceID   string
	AmountCents int64
	DueDate     time.Time
}

type ProcessBillingRequest struct {
	repository         repositories.PaymentRepository
	gateway            ports.PaymentGateway
	transactionManager ports.TransactionManager
	bus                sharedevent.EventBus
	now                func() time.Time
}

func NewProcessBillingRequest(
	repository repositories.PaymentRepository,
	gateway ports.PaymentGateway,
	transactionManager ports.TransactionManager,
	bus sharedevent.EventBus,
) ProcessBillingRequest {
	return ProcessBillingRequest{
		repository:         repository,
		gateway:            gateway,
		transactionManager: transactionManager,
		bus:                bus,
		now:                time.Now,
	}
}

func (usecase ProcessBillingRequest) Execute(ctx context.Context, input ProcessBillingRequestInput) (dto.Payment, error) {
	billingID, err := uuid.Parse(input.BillingID)
	if err != nil {
		return dto.Payment{}, entities.ErrInvalidBillingReference
	}

	invoiceID, err := uuid.Parse(input.InvoiceID)
	if err != nil {
		return dto.Payment{}, entities.ErrInvalidInvoiceReference
	}

	var payment entities.Payment
	err = usecase.transactionManager.WithinTransaction(ctx, func(txContext context.Context) error {
		hasApproved, err := usecase.repository.HasApprovedByInvoiceID(txContext, invoiceID.String())
		if err != nil {
			return fmt.Errorf("check approved payment: %w", err)
		}
		if hasApproved {
			return entities.ErrPaymentAlreadyExists
		}

		payment, err = entities.NewPayment(uuid.NewString(), billingID.String(), invoiceID.String(), usecase.now())
		if err != nil {
			return err
		}

		result, err := usecase.gateway.Process(txContext, ports.GatewayRequest{
			BillingID:   billingID.String(),
			InvoiceID:   invoiceID.String(),
			AmountCents: input.AmountCents,
			DueDate:     input.DueDate,
		})
		if err != nil {
			return fmt.Errorf("process payment: %w", err)
		}

		if result.Status == string(entities.StatusApproved) {
			payment.MarkApproved(result.GatewayReference, usecase.now())
		} else {
			payment.MarkFailed(result.GatewayReference, usecase.now())
		}

		if err := usecase.repository.Save(txContext, payment); err != nil {
			return fmt.Errorf("save payment: %w", err)
		}

		if payment.Status() == entities.StatusApproved {
			return sharedevent.Publish(txContext, usecase.bus, "payments", paymentevents.PaymentApproved{
				PaymentID:        payment.ID(),
				BillingID:        payment.BillingID(),
				InvoiceID:        payment.InvoiceID(),
				GatewayReference: payment.GatewayReference(),
				ApprovedAt:       payment.UpdatedAt(),
			})
		}

		return sharedevent.Publish(txContext, usecase.bus, "payments", paymentevents.PaymentFailed{
			PaymentID:        payment.ID(),
			BillingID:        payment.BillingID(),
			InvoiceID:        payment.InvoiceID(),
			GatewayReference: payment.GatewayReference(),
			FailedAt:         payment.UpdatedAt(),
		})
	})
	if err != nil {
		return dto.Payment{}, err
	}

	return mappers.ToPaymentDTO(payment), nil
}
