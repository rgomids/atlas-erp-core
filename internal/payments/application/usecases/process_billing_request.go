package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/rgomids/atlas-erp-core/internal/payments/application/dto"
	"github.com/rgomids/atlas-erp-core/internal/payments/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
	paymentevents "github.com/rgomids/atlas-erp-core/internal/payments/domain/events"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/repositories"
	"github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/mappers"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
)

type ProcessBillingRequestInput struct {
	BillingID     string
	InvoiceID     string
	CustomerID    string
	AmountCents   int64
	DueDate       time.Time
	AttemptNumber int
}

type ProcessBillingRequest struct {
	repository         repositories.PaymentRepository
	gateway            ports.PaymentGateway
	transactionManager ports.TransactionManager
	bus                sharedevent.EventBus
	gatewayTimeout     time.Duration
	now                func() time.Time
	observability      *observability.Runtime
}

func NewProcessBillingRequest(
	repository repositories.PaymentRepository,
	gateway ports.PaymentGateway,
	transactionManager ports.TransactionManager,
	bus sharedevent.EventBus,
	gatewayTimeout time.Duration,
	telemetry ...*observability.Runtime,
) ProcessBillingRequest {
	if gatewayTimeout <= 0 {
		gatewayTimeout = 2 * time.Second
	}

	return ProcessBillingRequest{
		repository:         repository,
		gateway:            gateway,
		transactionManager: transactionManager,
		bus:                bus,
		gatewayTimeout:     gatewayTimeout,
		now:                time.Now,
		observability:      observability.FromOptional(telemetry...),
	}
}

func (usecase ProcessBillingRequest) Execute(ctx context.Context, input ProcessBillingRequestInput) (paymentDTO dto.Payment, err error) {
	errorType := ""
	ctx, span := usecase.observability.StartUseCase(
		ctx,
		"payments",
		"ProcessBillingRequest",
		attribute.String("atlas.billing_id", input.BillingID),
		attribute.String("atlas.invoice_id", input.InvoiceID),
		attribute.String("atlas.customer_id", input.CustomerID),
		attribute.Int("atlas.attempt_number", input.AttemptNumber),
	)
	defer func() {
		usecase.observability.CompleteSpan(span, err, errorType)
	}()

	billingID, err := uuid.Parse(input.BillingID)
	if err != nil {
		errorType = observability.ErrorTypeValidation
		return dto.Payment{}, entities.ErrInvalidBillingReference
	}

	invoiceID, err := uuid.Parse(input.InvoiceID)
	if err != nil {
		errorType = observability.ErrorTypeValidation
		return dto.Payment{}, entities.ErrInvalidInvoiceReference
	}
	if _, err := uuid.Parse(input.CustomerID); err != nil {
		errorType = observability.ErrorTypeValidation
		return dto.Payment{}, entities.ErrInvalidCustomerReference
	}
	if input.AttemptNumber <= 0 {
		errorType = observability.ErrorTypeValidation
		return dto.Payment{}, entities.ErrInvalidAttemptNumber
	}

	idempotencyKey := buildIdempotencyKey(billingID.String(), input.AttemptNumber)

	var payment entities.Payment
	err = usecase.transactionManager.WithinTransaction(ctx, func(txContext context.Context) error {
		existing, err := usecase.repository.GetByBillingIDAndAttempt(txContext, billingID.String(), input.AttemptNumber)
		switch {
		case err == nil:
			payment = existing
			return nil
		case !errors.Is(err, entities.ErrPaymentNotFound):
			errorType = observability.ErrorTypeInfrastructure
			return fmt.Errorf("get payment by billing attempt: %w", err)
		}

		hasApproved, err := usecase.repository.HasApprovedByInvoiceID(txContext, invoiceID.String())
		if err != nil {
			errorType = observability.ErrorTypeInfrastructure
			return fmt.Errorf("check approved payment: %w", err)
		}
		if hasApproved {
			errorType = observability.ErrorTypeDomain
			return entities.ErrPaymentAlreadyExists
		}

		payment, err = entities.NewPayment(
			uuid.NewString(),
			billingID.String(),
			invoiceID.String(),
			input.AttemptNumber,
			idempotencyKey,
			usecase.now(),
		)
		if err != nil {
			errorType = observability.ErrorTypeDomain
			return err
		}

		if err := usecase.repository.Save(txContext, payment); err != nil {
			if errors.Is(err, entities.ErrPaymentAlreadyExists) {
				existing, getErr := usecase.repository.GetByBillingIDAndAttempt(txContext, billingID.String(), input.AttemptNumber)
				if getErr != nil {
					errorType = observability.ErrorTypeInfrastructure
					return fmt.Errorf("reload concurrent payment: %w", getErr)
				}

				payment = existing
				return nil
			}

			errorType = observability.ErrorTypeInfrastructure
			return fmt.Errorf("save payment: %w", err)
		}

		if input.AttemptNumber > 1 {
			usecase.observability.RecordPaymentRetry(txContext)
		}

		gatewayContext, cancel := context.WithTimeout(txContext, usecase.gatewayTimeout)
		defer cancel()

		startedAt := time.Now()
		gatewayContext, gatewaySpan := usecase.observability.StartIntegration(
			gatewayContext,
			"integration.gateway payments.Process",
			attribute.String("atlas.billing_id", billingID.String()),
			attribute.String("atlas.invoice_id", invoiceID.String()),
			attribute.Int("atlas.attempt_number", input.AttemptNumber),
		)

		result, err := usecase.gateway.Process(gatewayContext, ports.GatewayRequest{
			BillingID:   billingID.String(),
			InvoiceID:   invoiceID.String(),
			AmountCents: input.AmountCents,
			DueDate:     input.DueDate,
		})
		usecase.observability.RecordGatewayRequest(gatewayContext, time.Since(startedAt))
		if err != nil {
			errorType = observability.ErrorTypeIntegration
			usecase.observability.RecordGatewayFailure(gatewayContext, observability.ErrorTypeIntegration)
			payment.MarkFailed("", classifyGatewayError(err), usecase.now())
			if err := usecase.repository.Update(txContext, payment); err != nil {
				usecase.observability.CompleteSpan(gatewaySpan, err, observability.ErrorTypeInfrastructure)
				errorType = observability.ErrorTypeInfrastructure
				return fmt.Errorf("update failed payment after gateway error: %w", err)
			}
			usecase.observability.CompleteSpan(gatewaySpan, err, observability.ErrorTypeIntegration)

			return sharedevent.Publish(txContext, usecase.bus, "payments", paymentevents.PaymentFailed{
				PaymentID:        payment.ID(),
				BillingID:        payment.BillingID(),
				InvoiceID:        payment.InvoiceID(),
				CustomerID:       input.CustomerID,
				AttemptNumber:    payment.AttemptNumber(),
				IdempotencyKey:   payment.IdempotencyKey(),
				FailureCategory:  string(payment.FailureCategory()),
				GatewayReference: payment.GatewayReference(),
				FailedAt:         payment.UpdatedAt(),
			})
		}

		if result.Status == string(entities.StatusApproved) {
			payment.MarkApproved(result.GatewayReference, usecase.now())
			usecase.observability.CompleteSpan(gatewaySpan, nil, "")
		} else {
			payment.MarkFailed(result.GatewayReference, entities.FailureCategoryGatewayDeclined, usecase.now())
			usecase.observability.RecordGatewayFailure(gatewayContext, observability.ErrorTypeIntegration)
			usecase.observability.CompleteSpan(gatewaySpan, errors.New("gateway declined"), observability.ErrorTypeIntegration)
		}

		if err := usecase.repository.Update(txContext, payment); err != nil {
			errorType = observability.ErrorTypeInfrastructure
			return fmt.Errorf("update payment: %w", err)
		}

		if payment.Status() == entities.StatusApproved {
			return sharedevent.Publish(txContext, usecase.bus, "payments", paymentevents.PaymentApproved{
				PaymentID:        payment.ID(),
				BillingID:        payment.BillingID(),
				InvoiceID:        payment.InvoiceID(),
				CustomerID:       input.CustomerID,
				AttemptNumber:    payment.AttemptNumber(),
				IdempotencyKey:   payment.IdempotencyKey(),
				GatewayReference: payment.GatewayReference(),
				ApprovedAt:       payment.UpdatedAt(),
			})
		}

		errorType = observability.ErrorTypeIntegration
		return sharedevent.Publish(txContext, usecase.bus, "payments", paymentevents.PaymentFailed{
			PaymentID:        payment.ID(),
			BillingID:        payment.BillingID(),
			InvoiceID:        payment.InvoiceID(),
			CustomerID:       input.CustomerID,
			AttemptNumber:    payment.AttemptNumber(),
			IdempotencyKey:   payment.IdempotencyKey(),
			FailureCategory:  string(payment.FailureCategory()),
			GatewayReference: payment.GatewayReference(),
			FailedAt:         payment.UpdatedAt(),
		})
	})
	if err != nil {
		return dto.Payment{}, err
	}

	span.SetAttributes(
		attribute.String("atlas.payment_id", payment.ID()),
		attribute.String("atlas.idempotency_key", payment.IdempotencyKey()),
	)

	return mappers.ToPaymentDTO(payment), nil
}

func buildIdempotencyKey(billingID string, attemptNumber int) string {
	return fmt.Sprintf("billing:%s:attempt:%d", billingID, attemptNumber)
}

func classifyGatewayError(err error) entities.FailureCategory {
	if errors.Is(err, context.DeadlineExceeded) {
		return entities.FailureCategoryGatewayTimeout
	}

	return entities.FailureCategoryGatewayError
}
