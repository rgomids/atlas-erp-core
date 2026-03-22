package billing

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rgomids/atlas-erp-core/internal/billing/application/handlers"
	"github.com/rgomids/atlas-erp-core/internal/billing/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/billing/application/usecases"
	billingevents "github.com/rgomids/atlas-erp-core/internal/billing/domain/events"
	"github.com/rgomids/atlas-erp-core/internal/billing/infrastructure/persistence"
	invoiceevents "github.com/rgomids/atlas-erp-core/internal/invoices/domain/events"
	paymentevents "github.com/rgomids/atlas-erp-core/internal/payments/domain/events"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
)

type Module struct {
	paymentPort ports.PaymentCompatibilityPort
}

func NewModule(pool *pgxpool.Pool, bus sharedevent.EventBus, telemetry ...*observability.Runtime) Module {
	repository := persistence.NewPostgresRepository(pool)
	transactionManager := sharedpostgres.NewTxManager(pool)
	obs := observability.FromOptional(telemetry...)

	createBilling := usecases.NewCreateBillingFromInvoice(repository, bus, obs)
	getProcessableBilling := usecases.NewGetProcessableBillingByInvoiceID(repository, transactionManager, obs)
	markBillingApproved := usecases.NewMarkBillingApproved(repository, obs)
	markBillingFailed := usecases.NewMarkBillingFailed(repository, obs)

	sharedevent.Subscribe(bus, invoiceevents.InvoiceCreated{}.Name(), "billing", handlers.NewInvoiceCreated(createBilling))
	sharedevent.Subscribe(bus, paymentevents.PaymentApproved{}.Name(), "billing", handlers.NewPaymentApproved(markBillingApproved))
	sharedevent.Subscribe(bus, paymentevents.PaymentFailed{}.Name(), "billing", handlers.NewPaymentFailed(markBillingFailed))

	return Module{
		paymentPort: paymentPort{
			getProcessableBilling: getProcessableBilling,
		},
	}
}

func (module Module) PaymentPort() ports.PaymentCompatibilityPort {
	return module.paymentPort
}

type paymentPort struct {
	getProcessableBilling usecases.GetProcessableBillingByInvoiceID
}

func (port paymentPort) GetProcessableBillingByInvoiceID(ctx context.Context, invoiceID string) (ports.BillingSnapshot, error) {
	return port.getProcessableBilling.Execute(ctx, invoiceID)
}

func (Module) EventNames() []string {
	return []string{
		billingevents.BillingRequested{}.Name(),
	}
}
