package integration_test

import (
	"context"
	"testing"

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
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
	"github.com/rgomids/atlas-erp-core/test/support"
)

func TestPhase1FlowWithRealPostgres(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer pool.Close()

	customerRepository := customerpersistence.NewPostgresRepository(pool)
	createCustomer := customersusecases.NewCreateCustomer(customerRepository)

	customer, err := createCustomer.Execute(ctx, customersusecases.CreateCustomerInput{
		Name:     "Atlas Co",
		Document: "12345678900",
		Email:    "team@atlas.io",
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	customerModule := customers.NewModule(pool)
	invoiceRepository := invoicepersistence.NewPostgresRepository(pool)
	createInvoice := invoicesusecases.NewCreateInvoice(invoiceRepository, customerModule.ExistenceChecker())

	invoice, err := createInvoice.Execute(ctx, invoicesusecases.CreateInvoiceInput{
		CustomerID:  customer.ID,
		AmountCents: 2599,
		DueDate:     "2026-03-25",
	})
	if err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	invoiceModule := invoices.NewModule(pool, customerModule.ExistenceChecker())
	paymentRepository := paymentpersistence.NewPostgresRepository(pool)
	processPayment := paymentsusecases.NewProcessPayment(
		paymentRepository,
		invoiceModule.PaymentPort(),
		integration.NewMockGateway(),
		sharedpostgres.NewTxManager(pool),
	)

	payment, err := processPayment.Execute(ctx, paymentsusecases.ProcessPaymentInput{InvoiceID: invoice.ID})
	if err != nil {
		t.Fatalf("process payment: %v", err)
	}

	if payment.Status != "Approved" {
		t.Fatalf("expected approved payment, got %q", payment.Status)
	}

	storedInvoice, err := invoiceRepository.GetByID(ctx, invoice.ID)
	if err != nil {
		t.Fatalf("get stored invoice: %v", err)
	}

	if storedInvoice.Status() != "Paid" {
		t.Fatalf("expected paid invoice, got %q", storedInvoice.Status())
	}

	storedPayment, err := paymentRepository.GetByID(ctx, payment.ID)
	if err != nil {
		t.Fatalf("get stored payment: %v", err)
	}

	if storedPayment.Status() != "Approved" {
		t.Fatalf("expected approved stored payment, got %q", storedPayment.Status())
	}

	customerInvoices, err := invoiceRepository.ListByCustomerID(ctx, customer.ID)
	if err != nil {
		t.Fatalf("list customer invoices: %v", err)
	}

	if len(customerInvoices) != 1 {
		t.Fatalf("expected one invoice, got %d", len(customerInvoices))
	}
}

func TestPhase1RejectsDuplicatePaymentsWithRealPostgres(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer pool.Close()

	customerModule := customers.NewModule(pool)
	invoiceModule := invoices.NewModule(pool, customerModule.ExistenceChecker())
	_ = payments.NewModule(pool, invoiceModule.PaymentPort())

	customerRepository := customerpersistence.NewPostgresRepository(pool)
	createCustomer := customersusecases.NewCreateCustomer(customerRepository)
	customer, err := createCustomer.Execute(ctx, customersusecases.CreateCustomerInput{
		Name:     "Atlas Co",
		Document: "98765432100",
		Email:    "finance@atlas.io",
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	invoiceRepository := invoicepersistence.NewPostgresRepository(pool)
	createInvoice := invoicesusecases.NewCreateInvoice(invoiceRepository, customerModule.ExistenceChecker())
	invoice, err := createInvoice.Execute(ctx, invoicesusecases.CreateInvoiceInput{
		CustomerID:  customer.ID,
		AmountCents: 1099,
		DueDate:     "2026-03-27",
	})
	if err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	paymentRepository := paymentpersistence.NewPostgresRepository(pool)
	processPayment := paymentsusecases.NewProcessPayment(
		paymentRepository,
		invoiceModule.PaymentPort(),
		integration.NewMockGateway(),
		sharedpostgres.NewTxManager(pool),
	)

	if _, err := processPayment.Execute(ctx, paymentsusecases.ProcessPaymentInput{InvoiceID: invoice.ID}); err != nil {
		t.Fatalf("process first payment: %v", err)
	}

	if _, err := processPayment.Execute(ctx, paymentsusecases.ProcessPaymentInput{InvoiceID: invoice.ID}); err == nil {
		t.Fatal("expected duplicate payment error, got nil")
	}
}
