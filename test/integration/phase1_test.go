package integration_test

import (
	"context"
	"testing"

	"github.com/rgomids/atlas-erp-core/internal/billing"
	billingpersistence "github.com/rgomids/atlas-erp-core/internal/billing/infrastructure/persistence"
	"github.com/rgomids/atlas-erp-core/internal/customers"
	customersusecases "github.com/rgomids/atlas-erp-core/internal/customers/application/usecases"
	customerpersistence "github.com/rgomids/atlas-erp-core/internal/customers/infrastructure/persistence"
	"github.com/rgomids/atlas-erp-core/internal/invoices"
	invoicesusecases "github.com/rgomids/atlas-erp-core/internal/invoices/application/usecases"
	invoicepersistence "github.com/rgomids/atlas-erp-core/internal/invoices/infrastructure/persistence"
	"github.com/rgomids/atlas-erp-core/internal/payments"
	paymentsusecases "github.com/rgomids/atlas-erp-core/internal/payments/application/usecases"
	"github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/integration"
	paymentpersistence "github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/persistence"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
	"github.com/rgomids/atlas-erp-core/test/support"
)

func TestPhase3FlowWithRealPostgres(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer pool.Close()

	eventBus := sharedevent.NewSyncBus()
	customerModule := customers.NewModule(pool, eventBus)
	invoices.NewModule(pool, customerModule.ExistenceChecker(), eventBus)
	billingModule := billing.NewModule(pool, eventBus)
	_ = payments.NewModule(pool, billingModule.PaymentPort(), eventBus, integration.NewMockGateway())

	customerRepository := customerpersistence.NewPostgresRepository(pool)
	createCustomer := customersusecases.NewCreateCustomer(customerRepository, eventBus)

	customer, err := createCustomer.Execute(ctx, customersusecases.CreateCustomerInput{
		Name:     "Atlas Co",
		Document: "12345678900",
		Email:    "team@atlas.io",
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	invoiceRepository := invoicepersistence.NewPostgresRepository(pool)
	createInvoice := invoicesusecases.NewCreateInvoice(invoiceRepository, customerModule.ExistenceChecker(), eventBus)

	invoice, err := createInvoice.Execute(ctx, invoicesusecases.CreateInvoiceInput{
		CustomerID:  customer.ID,
		AmountCents: 2599,
		DueDate:     "2026-03-25",
	})
	if err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	storedInvoice, err := invoiceRepository.GetByID(ctx, invoice.ID)
	if err != nil {
		t.Fatalf("get stored invoice: %v", err)
	}

	if storedInvoice.Status() != "Paid" {
		t.Fatalf("expected paid invoice, got %q", storedInvoice.Status())
	}

	billingRepository := billingpersistence.NewPostgresRepository(pool)
	storedBilling, err := billingRepository.GetByInvoiceID(ctx, invoice.ID)
	if err != nil {
		t.Fatalf("get stored billing: %v", err)
	}

	if storedBilling.Status() != "Approved" {
		t.Fatalf("expected approved billing, got %q", storedBilling.Status())
	}

	paymentRepository := paymentpersistence.NewPostgresRepository(pool)
	storedPayments, err := paymentRepository.ListByInvoiceID(ctx, invoice.ID)
	if err != nil {
		t.Fatalf("list payments by invoice: %v", err)
	}

	if len(storedPayments) != 1 {
		t.Fatalf("expected one payment, got %d", len(storedPayments))
	}

	if storedPayments[0].Status() != "Approved" {
		t.Fatalf("expected approved payment, got %q", storedPayments[0].Status())
	}
}

func TestPhase3FailureFlowKeepsInvoicePendingAndBillingFailed(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer pool.Close()

	eventBus := sharedevent.NewSyncBus()
	customerModule := customers.NewModule(pool, eventBus)
	invoices.NewModule(pool, customerModule.ExistenceChecker(), eventBus)
	billingModule := billing.NewModule(pool, eventBus)
	_ = payments.NewModule(pool, billingModule.PaymentPort(), eventBus, integration.NewMockGatewayWithStatus("Failed"))

	customerRepository := customerpersistence.NewPostgresRepository(pool)
	createCustomer := customersusecases.NewCreateCustomer(customerRepository, eventBus)
	customer, err := createCustomer.Execute(ctx, customersusecases.CreateCustomerInput{
		Name:     "Atlas Co",
		Document: "98765432100",
		Email:    "finance@atlas.io",
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	invoiceRepository := invoicepersistence.NewPostgresRepository(pool)
	createInvoice := invoicesusecases.NewCreateInvoice(invoiceRepository, customerModule.ExistenceChecker(), eventBus)
	invoice, err := createInvoice.Execute(ctx, invoicesusecases.CreateInvoiceInput{
		CustomerID:  customer.ID,
		AmountCents: 1099,
		DueDate:     "2026-03-27",
	})
	if err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	storedInvoice, err := invoiceRepository.GetByID(ctx, invoice.ID)
	if err != nil {
		t.Fatalf("get stored invoice: %v", err)
	}

	if storedInvoice.Status() != "Pending" {
		t.Fatalf("expected pending invoice after failed payment, got %q", storedInvoice.Status())
	}

	billingRepository := billingpersistence.NewPostgresRepository(pool)
	storedBilling, err := billingRepository.GetByInvoiceID(ctx, invoice.ID)
	if err != nil {
		t.Fatalf("get stored billing: %v", err)
	}

	if storedBilling.Status() != "Failed" {
		t.Fatalf("expected failed billing, got %q", storedBilling.Status())
	}

	paymentRepository := paymentpersistence.NewPostgresRepository(pool)
	storedPayments, err := paymentRepository.ListByInvoiceID(ctx, invoice.ID)
	if err != nil {
		t.Fatalf("list payments by invoice: %v", err)
	}

	if len(storedPayments) != 1 {
		t.Fatalf("expected one failed payment, got %d", len(storedPayments))
	}

	if storedPayments[0].Status() != "Failed" {
		t.Fatalf("expected failed payment, got %q", storedPayments[0].Status())
	}
}

func TestPhase3RetryAfterFailureCreatesSecondAttempt(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer pool.Close()

	failedBus := sharedevent.NewSyncBus()
	customerModule := customers.NewModule(pool, failedBus)
	invoices.NewModule(pool, customerModule.ExistenceChecker(), failedBus)
	billingModule := billing.NewModule(pool, failedBus)
	_ = payments.NewModule(pool, billingModule.PaymentPort(), failedBus, integration.NewMockGatewayWithStatus("Failed"))

	customerRepository := customerpersistence.NewPostgresRepository(pool)
	createCustomer := customersusecases.NewCreateCustomer(customerRepository, failedBus)
	customer, err := createCustomer.Execute(ctx, customersusecases.CreateCustomerInput{
		Name:     "Atlas Co",
		Document: "45678912300",
		Email:    "ops@atlas.io",
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	invoiceRepository := invoicepersistence.NewPostgresRepository(pool)
	createInvoice := invoicesusecases.NewCreateInvoice(invoiceRepository, customerModule.ExistenceChecker(), failedBus)
	invoice, err := createInvoice.Execute(ctx, invoicesusecases.CreateInvoiceInput{
		CustomerID:  customer.ID,
		AmountCents: 3200,
		DueDate:     "2026-03-29",
	})
	if err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	retryBus := sharedevent.NewSyncBus()
	retryCustomerModule := customers.NewModule(pool, retryBus)
	invoices.NewModule(pool, retryCustomerModule.ExistenceChecker(), retryBus)
	retryBillingModule := billing.NewModule(pool, retryBus)

	paymentRepository := paymentpersistence.NewPostgresRepository(pool)
	processBillingRequest := paymentsusecases.NewProcessBillingRequest(
		paymentRepository,
		integration.NewMockGateway(),
		sharedpostgres.NewTxManager(pool),
		retryBus,
	)
	processPayment := paymentsusecases.NewProcessPayment(retryBillingModule.PaymentPort(), processBillingRequest)

	payment, err := processPayment.Execute(ctx, paymentsusecases.ProcessPaymentInput{InvoiceID: invoice.ID})
	if err != nil {
		t.Fatalf("retry payment: %v", err)
	}

	if payment.Status != "Approved" {
		t.Fatalf("expected approved retry payment, got %q", payment.Status)
	}

	storedInvoice, err := invoiceRepository.GetByID(ctx, invoice.ID)
	if err != nil {
		t.Fatalf("get stored invoice: %v", err)
	}

	if storedInvoice.Status() != "Paid" {
		t.Fatalf("expected paid invoice after retry, got %q", storedInvoice.Status())
	}

	storedPayments, err := paymentRepository.ListByInvoiceID(ctx, invoice.ID)
	if err != nil {
		t.Fatalf("list payments by invoice: %v", err)
	}

	if len(storedPayments) != 2 {
		t.Fatalf("expected two payment attempts, got %d", len(storedPayments))
	}

	approvedCount := 0
	for _, storedPayment := range storedPayments {
		if storedPayment.Status() == "Approved" {
			approvedCount++
		}
	}

	if approvedCount != 1 {
		t.Fatalf("expected exactly one approved payment, got %d", approvedCount)
	}
}
