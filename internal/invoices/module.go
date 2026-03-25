package invoices

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	customerpublic "github.com/rgomids/atlas-erp-core/internal/customers/public"
	"github.com/rgomids/atlas-erp-core/internal/invoices/application/handlers"
	"github.com/rgomids/atlas-erp-core/internal/invoices/application/usecases"
	invoiceshttp "github.com/rgomids/atlas-erp-core/internal/invoices/infrastructure/http"
	"github.com/rgomids/atlas-erp-core/internal/invoices/infrastructure/persistence"
	paymentevents "github.com/rgomids/atlas-erp-core/internal/payments/public/events"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
)

type Module struct {
	handler invoiceshttp.Handler
}

func NewModule(pool *pgxpool.Pool, customerExistenceChecker customerpublic.ExistenceChecker, bus sharedevent.EventBus, telemetry ...*observability.Runtime) Module {
	repository := persistence.NewPostgresRepository(pool)
	obs := observability.FromOptional(telemetry...)
	applyPaymentApproved := usecases.NewApplyPaymentApproved(repository, bus, obs)

	sharedevent.Subscribe(bus, paymentevents.EventNamePaymentApproved, "invoices", handlers.NewPaymentApproved(applyPaymentApproved))

	return Module{
		handler: invoiceshttp.NewHandler(
			usecases.NewCreateInvoice(repository, customerExistenceChecker, bus, obs),
			usecases.NewListCustomerInvoices(repository, obs),
		),
	}
}

func (module Module) Routes(router chi.Router) {
	module.handler.Routes(router)
}
