package invoices

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	customerports "github.com/rgomids/atlas-erp-core/internal/customers/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/invoices/application/handlers"
	"github.com/rgomids/atlas-erp-core/internal/invoices/application/usecases"
	invoiceshttp "github.com/rgomids/atlas-erp-core/internal/invoices/infrastructure/http"
	"github.com/rgomids/atlas-erp-core/internal/invoices/infrastructure/persistence"
	paymentevents "github.com/rgomids/atlas-erp-core/internal/payments/domain/events"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

type Module struct {
	handler invoiceshttp.Handler
}

func NewModule(pool *pgxpool.Pool, customerExistenceChecker customerports.ExistenceChecker, bus sharedevent.EventBus) Module {
	repository := persistence.NewPostgresRepository(pool)
	applyPaymentApproved := usecases.NewApplyPaymentApproved(repository, bus)

	sharedevent.Subscribe(bus, paymentevents.PaymentApproved{}.Name(), "invoices", handlers.NewPaymentApproved(applyPaymentApproved))

	return Module{
		handler: invoiceshttp.NewHandler(
			usecases.NewCreateInvoice(repository, customerExistenceChecker, bus),
			usecases.NewListCustomerInvoices(repository),
		),
	}
}

func (module Module) Routes(router chi.Router) {
	module.handler.Routes(router)
}
