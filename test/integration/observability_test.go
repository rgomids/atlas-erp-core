package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/atlas-erp-core/internal/billing"
	"github.com/rgomids/atlas-erp-core/internal/customers"
	customerusecases "github.com/rgomids/atlas-erp-core/internal/customers/application/usecases"
	customerpersistence "github.com/rgomids/atlas-erp-core/internal/customers/infrastructure/persistence"
	"github.com/rgomids/atlas-erp-core/internal/invoices"
	invoicesusecases "github.com/rgomids/atlas-erp-core/internal/invoices/application/usecases"
	invoicepersistence "github.com/rgomids/atlas-erp-core/internal/invoices/infrastructure/persistence"
	"github.com/rgomids/atlas-erp-core/internal/payments"
	paymentsusecases "github.com/rgomids/atlas-erp-core/internal/payments/application/usecases"
	"github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/integration"
	paymentpersistence "github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/persistence"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
	"github.com/rgomids/atlas-erp-core/internal/shared/outbox"
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
	"github.com/rgomids/atlas-erp-core/test/support"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestPhase5ObservabilityTracesAndMetricsCriticalFlow(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	exporter := tracetest.NewInMemoryExporter()
	telemetry, err := observability.New(ctx, observability.Config{
		ServiceName:   "atlas-erp-core",
		Environment:   "test",
		TraceExporter: exporter,
	})
	if err != nil {
		t.Fatalf("create telemetry runtime: %v", err)
	}

	pool, err := sharedpostgres.Open(ctx, databaseConfig, telemetry)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer pool.Close()

	eventBus := sharedevent.NewSyncBusWithObservability(telemetry, outbox.NewPostgresRecorder(pool))
	customerModule := customers.NewModule(pool, eventBus, telemetry)
	invoices.NewModule(pool, customerModule.ExistenceChecker(), eventBus, telemetry)
	billingModule := billing.NewModule(pool, eventBus, telemetry)
	_ = payments.NewModule(pool, billingModule.PaymentPort(), eventBus, integration.NewMockGateway(), payments.ModuleConfig{
		GatewayTimeout: time.Second,
	}, telemetry)

	customerRepository := customerpersistence.NewPostgresRepository(pool)
	createCustomer := customerusecases.NewCreateCustomer(customerRepository, eventBus, telemetry)
	customer, err := createCustomer.Execute(ctx, customerusecases.CreateCustomerInput{
		Name:     "Atlas Co",
		Document: "32165498700",
		Email:    "trace@atlas.io",
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	invoiceRepository := invoicepersistence.NewPostgresRepository(pool)
	createInvoice := invoicesusecases.NewCreateInvoice(invoiceRepository, customerModule.ExistenceChecker(), eventBus, telemetry)
	if _, err := createInvoice.Execute(ctx, invoicesusecases.CreateInvoiceInput{
		CustomerID:  customer.ID,
		AmountCents: 4100,
		DueDate:     "2026-03-25",
	}); err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	spans := exporter.GetSpans()
	expectedSpans := []string{
		"application.usecase customers.CreateCustomer",
		"application.usecase invoices.CreateInvoice",
		"application.usecase billing.CreateBillingFromInvoice",
		"application.usecase payments.ProcessBillingRequest",
		"event.publish InvoiceCreated",
		"event.consume billing.InvoiceCreated",
		"integration.gateway payments.Process",
	}
	for _, expectedSpan := range expectedSpans {
		if !integrationContainsSpan(spans, expectedSpan) {
			t.Fatalf("expected spans to contain %s, got %#v", expectedSpan, spans)
		}
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	telemetry.MetricsHandler().ServeHTTP(response, request)

	metricBody := response.Body.String()
	for _, fragment := range []string{
		"atlas_erp_events_published_total",
		"atlas_erp_db_query_duration_seconds",
		"atlas_erp_gateway_request_duration_seconds",
	} {
		if !strings.Contains(metricBody, fragment) {
			t.Fatalf("expected metrics output to contain %s, got %s", fragment, metricBody)
		}
	}
}

func TestPhase5ObservabilityRetryMetricIncrementsOnSecondAttempt(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	exporter := tracetest.NewInMemoryExporter()
	telemetry, err := observability.New(ctx, observability.Config{
		ServiceName:   "atlas-erp-core",
		Environment:   "test",
		TraceExporter: exporter,
	})
	if err != nil {
		t.Fatalf("create telemetry runtime: %v", err)
	}

	pool, err := sharedpostgres.Open(ctx, databaseConfig, telemetry)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer pool.Close()

	failedBus := sharedevent.NewSyncBusWithObservability(telemetry, outbox.NewPostgresRecorder(pool))
	failedCustomerModule := customers.NewModule(pool, failedBus, telemetry)
	invoices.NewModule(pool, failedCustomerModule.ExistenceChecker(), failedBus, telemetry)
	failedBillingModule := billing.NewModule(pool, failedBus, telemetry)
	_ = payments.NewModule(pool, failedBillingModule.PaymentPort(), failedBus, integration.NewMockGatewayWithStatus("Failed"), payments.ModuleConfig{
		GatewayTimeout: time.Second,
	}, telemetry)

	customerRepository := customerpersistence.NewPostgresRepository(pool)
	createCustomer := customerusecases.NewCreateCustomer(customerRepository, failedBus, telemetry)
	customer, err := createCustomer.Execute(ctx, customerusecases.CreateCustomerInput{
		Name:     "Atlas Co",
		Document: "74185296300",
		Email:    "retry@atlas.io",
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	invoiceRepository := invoicepersistence.NewPostgresRepository(pool)
	createInvoice := invoicesusecases.NewCreateInvoice(invoiceRepository, failedCustomerModule.ExistenceChecker(), failedBus, telemetry)
	invoice, err := createInvoice.Execute(ctx, invoicesusecases.CreateInvoiceInput{
		CustomerID:  customer.ID,
		AmountCents: 5500,
		DueDate:     "2026-03-27",
	})
	if err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	retryBus := sharedevent.NewSyncBusWithObservability(telemetry, outbox.NewPostgresRecorder(pool))
	retryCustomerModule := customers.NewModule(pool, retryBus, telemetry)
	invoices.NewModule(pool, retryCustomerModule.ExistenceChecker(), retryBus, telemetry)
	retryBillingModule := billing.NewModule(pool, retryBus, telemetry)
	paymentRepository := paymentpersistence.NewPostgresRepository(pool)
	processBillingRequest := paymentsusecases.NewProcessBillingRequest(
		paymentRepository,
		integration.NewMockGateway(),
		sharedpostgres.NewTxManager(pool),
		retryBus,
		time.Second,
		telemetry,
	)
	processPayment := paymentsusecases.NewProcessPayment(retryBillingModule.PaymentPort(), processBillingRequest, telemetry)
	if _, err := processPayment.Execute(ctx, paymentsusecases.ProcessPaymentInput{InvoiceID: invoice.ID}); err != nil {
		t.Fatalf("retry payment: %v", err)
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	telemetry.MetricsHandler().ServeHTTP(response, request)

	if !strings.Contains(response.Body.String(), "atlas_erp_payment_retries_total") {
		t.Fatalf("expected metrics output to include atlas_erp_payment_retries_total, got %s", response.Body.String())
	}

	_ = invoice
}

func integrationContainsSpan(spans tracetest.SpanStubs, expectedName string) bool {
	for _, span := range spans {
		if span.Name == expectedName {
			return true
		}
	}

	return false
}
