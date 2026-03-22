package payments

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	invoiceports "github.com/rgomids/atlas-erp-core/internal/invoices/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/payments/application/usecases"
	paymentshttp "github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/http"
	"github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/integration"
	"github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/persistence"
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
)

type Module struct {
	handler paymentshttp.Handler
}

func NewModule(pool *pgxpool.Pool, invoicePaymentPort invoiceports.InvoicePaymentPort) Module {
	repository := persistence.NewPostgresRepository(pool)

	return Module{
		handler: paymentshttp.NewHandler(
			usecases.NewProcessPayment(
				repository,
				invoicePaymentPort,
				integration.NewMockGateway(),
				sharedpostgres.NewTxManager(pool),
			),
		),
	}
}

func (module Module) Routes(router chi.Router) {
	module.handler.Routes(router)
}
