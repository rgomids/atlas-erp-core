package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	invoiceports "github.com/rgomids/atlas-erp-core/internal/invoices/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/payments/application/dto"
	"github.com/rgomids/atlas-erp-core/internal/payments/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/repositories"
	"github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/mappers"
)

type ProcessPaymentInput struct {
	InvoiceID string
}

type ProcessPayment struct {
	repository         repositories.PaymentRepository
	invoicePaymentPort invoiceports.InvoicePaymentPort
	gateway            ports.PaymentGateway
	transactionManager ports.TransactionManager
	now                func() time.Time
}

func NewProcessPayment(
	repository repositories.PaymentRepository,
	invoicePaymentPort invoiceports.InvoicePaymentPort,
	gateway ports.PaymentGateway,
	transactionManager ports.TransactionManager,
) ProcessPayment {
	return ProcessPayment{
		repository:         repository,
		invoicePaymentPort: invoicePaymentPort,
		gateway:            gateway,
		transactionManager: transactionManager,
		now:                time.Now,
	}
}

func (usecase ProcessPayment) Execute(ctx context.Context, input ProcessPaymentInput) (dto.Payment, error) {
	invoiceID, err := uuid.Parse(input.InvoiceID)
	if err != nil {
		return dto.Payment{}, entities.ErrInvalidInvoiceReference
	}

	var payment entities.Payment
	err = usecase.transactionManager.WithinTransaction(ctx, func(txContext context.Context) error {
		exists, err := usecase.repository.ExistsByInvoiceID(txContext, invoiceID.String())
		if err != nil {
			return fmt.Errorf("check payment duplication: %w", err)
		}
		if exists {
			return entities.ErrPaymentAlreadyExists
		}

		invoice, err := usecase.invoicePaymentPort.GetPayableInvoice(txContext, invoiceID.String())
		if err != nil {
			return err
		}

		payment, err = entities.NewPayment(uuid.NewString(), invoiceID.String(), usecase.now())
		if err != nil {
			return err
		}

		result, err := usecase.gateway.Process(txContext, ports.GatewayRequest{Invoice: invoice})
		if err != nil {
			return fmt.Errorf("process payment: %w", err)
		}

		switch result.Status {
		case string(entities.StatusApproved):
			payment.MarkApproved(result.GatewayReference, usecase.now())
		default:
			payment.MarkFailed(result.GatewayReference, usecase.now())
		}

		if err := usecase.repository.Save(txContext, payment); err != nil {
			return fmt.Errorf("save payment: %w", err)
		}

		if payment.Status() == entities.StatusApproved {
			if err := usecase.invoicePaymentPort.MarkAsPaid(txContext, invoiceID.String(), usecase.now()); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return dto.Payment{}, err
	}

	return mappers.ToPaymentDTO(payment), nil
}
