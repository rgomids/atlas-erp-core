package repositories

import (
	"context"

	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
)

type PaymentRepository interface {
	ExistsByInvoiceID(ctx context.Context, invoiceID string) (bool, error)
	Save(ctx context.Context, payment entities.Payment) error
	GetByID(ctx context.Context, paymentID string) (entities.Payment, error)
}
