package repositories

import (
	"context"

	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
)

type PaymentRepository interface {
	HasApprovedByInvoiceID(ctx context.Context, invoiceID string) (bool, error)
	Save(ctx context.Context, payment entities.Payment) error
	Update(ctx context.Context, payment entities.Payment) error
	GetByID(ctx context.Context, paymentID string) (entities.Payment, error)
	GetByBillingIDAndAttempt(ctx context.Context, billingID string, attemptNumber int) (entities.Payment, error)
	ListByInvoiceID(ctx context.Context, invoiceID string) ([]entities.Payment, error)
}
