package payments

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	billingpublic "github.com/rgomids/atlas-erp-core/internal/billing/public"
	billingevents "github.com/rgomids/atlas-erp-core/internal/billing/public/events"
	"github.com/rgomids/atlas-erp-core/internal/payments/application/handlers"
	"github.com/rgomids/atlas-erp-core/internal/payments/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/payments/application/usecases"
	paymentshttp "github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/http"
	"github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/integration"
	"github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/persistence"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
)

type Module struct {
	handler paymentshttp.Handler
}

type ModuleConfig struct {
	GatewayTimeout time.Duration
}

func NewModule(
	pool *pgxpool.Pool,
	billingPort billingpublic.PaymentCompatibilityPort,
	bus sharedevent.EventBus,
	gateway ports.PaymentGateway,
	config ModuleConfig,
	telemetry ...*observability.Runtime,
) Module {
	repository := persistence.NewPostgresRepository(pool)
	if gateway == nil {
		gateway = integration.NewMockGateway()
	}
	obs := observability.FromOptional(telemetry...)

	processBillingRequest := usecases.NewProcessBillingRequest(
		repository,
		gateway,
		sharedpostgres.NewTxManager(pool),
		bus,
		config.GatewayTimeout,
		obs,
	)
	sharedevent.Subscribe(bus, billingevents.EventNameBillingRequested, "payments", handlers.NewBillingRequested(processBillingRequest))

	return Module{
		handler: paymentshttp.NewHandler(
			usecases.NewProcessPayment(
				billingPort,
				processBillingRequest,
				obs,
			),
		),
	}
}

func (module Module) Routes(router chi.Router) {
	module.handler.Routes(router)
}
