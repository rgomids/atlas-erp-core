package billing

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rgomids/atlas-erp-core/internal/billing/application/handlers"
	"github.com/rgomids/atlas-erp-core/internal/billing/application/usecases"
	"github.com/rgomids/atlas-erp-core/internal/billing/infrastructure/persistence"
	billingpublic "github.com/rgomids/atlas-erp-core/internal/billing/public"
	billingevents "github.com/rgomids/atlas-erp-core/internal/billing/public/events"
	invoiceevents "github.com/rgomids/atlas-erp-core/internal/invoices/public/events"
	paymentevents "github.com/rgomids/atlas-erp-core/internal/payments/public/events"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
)

type Module struct {
	paymentPort billingpublic.PaymentCompatibilityPort
}

func NewModule(pool *pgxpool.Pool, bus sharedevent.EventBus, telemetry ...*observability.Runtime) Module {
	repository := persistence.NewPostgresRepository(pool)
	transactionManager := sharedpostgres.NewTxManager(pool)
	obs := observability.FromOptional(telemetry...)

	createBilling := usecases.NewCreateBillingFromInvoice(repository, bus, obs)
	getProcessableBilling := usecases.NewGetProcessableBillingByInvoiceID(repository, transactionManager, obs)
	markBillingApproved := usecases.NewMarkBillingApproved(repository, obs)
	markBillingFailed := usecases.NewMarkBillingFailed(repository, obs)

	sharedevent.Subscribe(bus, invoiceevents.EventNameInvoiceCreated, "billing", handlers.NewInvoiceCreated(createBilling))
	sharedevent.Subscribe(bus, paymentevents.EventNamePaymentApproved, "billing", handlers.NewPaymentApproved(markBillingApproved))
	sharedevent.Subscribe(bus, paymentevents.EventNamePaymentFailed, "billing", handlers.NewPaymentFailed(markBillingFailed))

	return Module{
		paymentPort: paymentPort{
			getProcessableBilling: getProcessableBilling,
		},
	}
}

func (module Module) PaymentPort() billingpublic.PaymentCompatibilityPort {
	return module.paymentPort
}

type paymentPort struct {
	getProcessableBilling usecases.GetProcessableBillingByInvoiceID
}

func (port paymentPort) GetProcessableBillingByInvoiceID(ctx context.Context, invoiceID string) (billingpublic.BillingSnapshot, error) {
	return port.getProcessableBilling.Execute(ctx, invoiceID)
}

func (Module) EventNames() []string {
	return []string{
		billingevents.EventNameBillingRequested,
	}
}
