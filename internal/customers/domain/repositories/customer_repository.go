package repositories

import (
	"context"

	"github.com/rgomids/atlas-erp-core/internal/customers/domain/entities"
)

type CustomerRepository interface {
	ExistsByDocument(ctx context.Context, document string) (bool, error)
	Save(ctx context.Context, customer entities.Customer) error
	GetByID(ctx context.Context, customerID string) (entities.Customer, error)
	Update(ctx context.Context, customer entities.Customer) error
}
