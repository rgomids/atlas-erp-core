package repositories

import (
	"context"

	"github.com/rgomids/atlas-erp-core/internal/billing/domain/entities"
)

type BillingRepository interface {
	Save(ctx context.Context, billing entities.Billing) error
	GetByID(ctx context.Context, billingID string) (entities.Billing, error)
	GetByInvoiceID(ctx context.Context, invoiceID string) (entities.Billing, error)
	GetByInvoiceIDForUpdate(ctx context.Context, invoiceID string) (entities.Billing, error)
	Update(ctx context.Context, billing entities.Billing) error
}
