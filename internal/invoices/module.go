package invoices

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	customerports "github.com/rgomids/atlas-erp-core/internal/customers/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/invoices/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/invoices/application/usecases"
	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/entities"
	invoiceshttp "github.com/rgomids/atlas-erp-core/internal/invoices/infrastructure/http"
	"github.com/rgomids/atlas-erp-core/internal/invoices/infrastructure/mappers"
	"github.com/rgomids/atlas-erp-core/internal/invoices/infrastructure/persistence"
)

type Module struct {
	handler     invoiceshttp.Handler
	paymentPort ports.InvoicePaymentPort
}

func NewModule(pool *pgxpool.Pool, customerExistenceChecker customerports.ExistenceChecker) Module {
	repository := persistence.NewPostgresRepository(pool)

	return Module{
		handler: invoiceshttp.NewHandler(
			usecases.NewCreateInvoice(repository, customerExistenceChecker),
			usecases.NewListCustomerInvoices(repository),
		),
		paymentPort: paymentPort{repository: repository},
	}
}

func (module Module) Routes(router chi.Router) {
	module.handler.Routes(router)
}

func (module Module) PaymentPort() ports.InvoicePaymentPort {
	return module.paymentPort
}

type paymentPort struct {
	repository *persistence.PostgresRepository
}

func (port paymentPort) GetPayableInvoice(ctx context.Context, invoiceID string) (ports.InvoiceSnapshot, error) {
	invoice, err := port.repository.GetByID(ctx, invoiceID)
	if err != nil {
		return ports.InvoiceSnapshot{}, err
	}

	if !invoice.IsPayable() {
		return ports.InvoiceSnapshot{}, entities.ErrInvoiceNotPayable
	}

	return mappers.ToSnapshot(invoice), nil
}

func (port paymentPort) MarkAsPaid(ctx context.Context, invoiceID string, paidAt time.Time) error {
	invoice, err := port.repository.GetByID(ctx, invoiceID)
	if err != nil {
		return err
	}

	if err := invoice.MarkPaid(paidAt); err != nil {
		return err
	}

	return port.repository.Update(ctx, invoice)
}
