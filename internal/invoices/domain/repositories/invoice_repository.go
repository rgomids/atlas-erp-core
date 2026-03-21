package repositories

import (
	"context"

	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/entities"
)

type InvoiceRepository interface {
	Save(ctx context.Context, invoice entities.Invoice) error
	GetByID(ctx context.Context, invoiceID string) (entities.Invoice, error)
	ListByCustomerID(ctx context.Context, customerID string) ([]entities.Invoice, error)
	Update(ctx context.Context, invoice entities.Invoice) error
}
