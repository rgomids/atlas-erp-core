package payments

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	billingports "github.com/rgomids/atlas-erp-core/internal/billing/application/ports"
	billingevents "github.com/rgomids/atlas-erp-core/internal/billing/domain/events"
	"github.com/rgomids/atlas-erp-core/internal/payments/application/handlers"
	"github.com/rgomids/atlas-erp-core/internal/payments/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/payments/application/usecases"
	paymentshttp "github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/http"
	"github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/integration"
	"github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/persistence"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
)

type Module struct {
	handler paymentshttp.Handler
}

func NewModule(
	pool *pgxpool.Pool,
	billingPort billingports.PaymentCompatibilityPort,
	bus sharedevent.EventBus,
	gateway ports.PaymentGateway,
) Module {
	repository := persistence.NewPostgresRepository(pool)
	if gateway == nil {
		gateway = integration.NewMockGateway()
	}

	processBillingRequest := usecases.NewProcessBillingRequest(
		repository,
		gateway,
		sharedpostgres.NewTxManager(pool),
		bus,
	)
	sharedevent.Subscribe(bus, billingevents.BillingRequested{}.Name(), "payments", handlers.NewBillingRequested(processBillingRequest))

	return Module{
		handler: paymentshttp.NewHandler(
			usecases.NewProcessPayment(
				billingPort,
				processBillingRequest,
			),
		),
	}
}

func (module Module) Routes(router chi.Router) {
	module.handler.Routes(router)
}
