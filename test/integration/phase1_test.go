package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rgomids/atlas-erp-core/internal/billing"
	billingpersistence "github.com/rgomids/atlas-erp-core/internal/billing/infrastructure/persistence"
	billingpublic "github.com/rgomids/atlas-erp-core/internal/billing/public"
	"github.com/rgomids/atlas-erp-core/internal/customers"
	customersusecases "github.com/rgomids/atlas-erp-core/internal/customers/application/usecases"
	customerentities "github.com/rgomids/atlas-erp-core/internal/customers/domain/entities"
	customerpersistence "github.com/rgomids/atlas-erp-core/internal/customers/infrastructure/persistence"
	customerevents "github.com/rgomids/atlas-erp-core/internal/customers/public/events"
	"github.com/rgomids/atlas-erp-core/internal/invoices"
	invoicesusecases "github.com/rgomids/atlas-erp-core/internal/invoices/application/usecases"
	invoicepersistence "github.com/rgomids/atlas-erp-core/internal/invoices/infrastructure/persistence"
	invoiceevents "github.com/rgomids/atlas-erp-core/internal/invoices/public/events"
	"github.com/rgomids/atlas-erp-core/internal/payments"
	paymentports "github.com/rgomids/atlas-erp-core/internal/payments/application/ports"
	paymentsusecases "github.com/rgomids/atlas-erp-core/internal/payments/application/usecases"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/integration"
	paymentpersistence "github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/persistence"
	"github.com/rgomids/atlas-erp-core/internal/shared/config"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	"github.com/rgomids/atlas-erp-core/internal/shared/outbox"
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
	"github.com/rgomids/atlas-erp-core/internal/shared/runtimefaults"
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

	eventBus := newIntegrationEventBus(pool)
	customerModule := customers.NewModule(pool, eventBus)
	invoices.NewModule(pool, customerModule.ExistenceChecker(), eventBus)
	billingModule := billing.NewModule(pool, eventBus)
	_ = payments.NewModule(pool, billingModule.PaymentPort(), eventBus, integration.NewMockGateway(), payments.ModuleConfig{
		GatewayTimeout: time.Second,
	})

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

	if count := countOutboxEvents(ctx, t, pool); count != 5 {
		t.Fatalf("expected 5 recorded outbox events, got %d", count)
	}

	for _, event := range listOutboxEvents(ctx, t, pool) {
		if event.Status != "processed" {
			t.Fatalf("expected all outbox events to be processed, got %q for %s", event.Status, event.EventName)
		}

		if event.AggregateID == "" {
			t.Fatalf("expected aggregate id to be set for %s", event.EventName)
		}

		if event.CorrelationID == "" {
			t.Fatalf("expected correlation id to be set for %s", event.EventName)
		}

		var payload struct {
			Metadata map[string]any `json:"metadata"`
			Payload  map[string]any `json:"payload"`
		}
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("decode outbox payload for %s: %v", event.EventName, err)
		}

		if payload.Metadata["event_name"] != event.EventName {
			t.Fatalf("expected payload metadata event_name %q, got %#v", event.EventName, payload.Metadata["event_name"])
		}

		if payload.Metadata["aggregate_id"] != event.AggregateID {
			t.Fatalf("expected payload aggregate id %q, got %#v", event.AggregateID, payload.Metadata["aggregate_id"])
		}
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

	eventBus := newIntegrationEventBus(pool)
	customerModule := customers.NewModule(pool, eventBus)
	invoices.NewModule(pool, customerModule.ExistenceChecker(), eventBus)
	billingModule := billing.NewModule(pool, eventBus)
	_ = payments.NewModule(pool, billingModule.PaymentPort(), eventBus, integration.NewMockGatewayWithStatus("Failed"), payments.ModuleConfig{
		GatewayTimeout: time.Second,
	})

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

	if storedPayments[0].FailureCategory() != entities.FailureCategoryGatewayDeclined {
		t.Fatalf("expected gateway_declined failure category, got %q", storedPayments[0].FailureCategory())
	}

	failedEvent := outboxEventByName(ctx, t, pool, "PaymentFailed")
	if failedEvent.Status != "processed" {
		t.Fatalf("expected PaymentFailed outbox event to be processed, got %q", failedEvent.Status)
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

	failedBus := newIntegrationEventBus(pool)
	customerModule := customers.NewModule(pool, failedBus)
	invoices.NewModule(pool, customerModule.ExistenceChecker(), failedBus)
	billingModule := billing.NewModule(pool, failedBus)
	_ = payments.NewModule(pool, billingModule.PaymentPort(), failedBus, integration.NewMockGatewayWithStatus("Failed"), payments.ModuleConfig{
		GatewayTimeout: time.Second,
	})

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

	retryBus := newIntegrationEventBus(pool)
	retryCustomerModule := customers.NewModule(pool, retryBus)
	invoices.NewModule(pool, retryCustomerModule.ExistenceChecker(), retryBus)
	retryBillingModule := billing.NewModule(pool, retryBus)

	paymentRepository := paymentpersistence.NewPostgresRepository(pool)
	processBillingRequest := paymentsusecases.NewProcessBillingRequest(
		paymentRepository,
		integration.NewMockGateway(),
		sharedpostgres.NewTxManager(pool),
		retryBus,
		time.Second,
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

func TestPhase4TechnicalGatewayFailurePersistsRetryableAttempt(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer pool.Close()

	eventBus := newIntegrationEventBus(pool)
	customerModule := customers.NewModule(pool, eventBus)
	invoices.NewModule(pool, customerModule.ExistenceChecker(), eventBus)
	billingModule := billing.NewModule(pool, eventBus)
	_ = payments.NewModule(pool, billingModule.PaymentPort(), eventBus, integration.NewMockGatewayWithDelay("Approved", 25*time.Millisecond), payments.ModuleConfig{
		GatewayTimeout: 5 * time.Millisecond,
	})

	customerRepository := customerpersistence.NewPostgresRepository(pool)
	createCustomer := customersusecases.NewCreateCustomer(customerRepository, eventBus)
	customer, err := createCustomer.Execute(ctx, customersusecases.CreateCustomerInput{
		Name:     "Atlas Co",
		Document: "11122233344",
		Email:    "timeout@atlas.io",
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	invoiceRepository := invoicepersistence.NewPostgresRepository(pool)
	createInvoice := invoicesusecases.NewCreateInvoice(invoiceRepository, customerModule.ExistenceChecker(), eventBus)
	invoice, err := createInvoice.Execute(ctx, invoicesusecases.CreateInvoiceInput{
		CustomerID:  customer.ID,
		AmountCents: 9900,
		DueDate:     "2026-03-30",
	})
	if err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	storedInvoice, err := invoiceRepository.GetByID(ctx, invoice.ID)
	if err != nil {
		t.Fatalf("get stored invoice: %v", err)
	}

	if storedInvoice.Status() != "Pending" {
		t.Fatalf("expected pending invoice after gateway timeout, got %q", storedInvoice.Status())
	}

	billingRepository := billingpersistence.NewPostgresRepository(pool)
	storedBilling, err := billingRepository.GetByInvoiceID(ctx, invoice.ID)
	if err != nil {
		t.Fatalf("get stored billing: %v", err)
	}

	if storedBilling.Status() != "Failed" {
		t.Fatalf("expected failed billing after gateway timeout, got %q", storedBilling.Status())
	}

	if storedBilling.AttemptNumber() != 1 {
		t.Fatalf("expected first attempt to remain recorded, got %d", storedBilling.AttemptNumber())
	}

	paymentRepository := paymentpersistence.NewPostgresRepository(pool)
	storedPayments, err := paymentRepository.ListByInvoiceID(ctx, invoice.ID)
	if err != nil {
		t.Fatalf("list payments by invoice: %v", err)
	}

	if len(storedPayments) != 1 {
		t.Fatalf("expected one failed payment attempt, got %d", len(storedPayments))
	}

	if storedPayments[0].FailureCategory() != entities.FailureCategoryGatewayTimeout {
		t.Fatalf("expected gateway_timeout failure category, got %q", storedPayments[0].FailureCategory())
	}
}

func TestPhase6OutboxMarksFailedWhenHandlerFailsAfterAppend(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer pool.Close()

	eventBus := newIntegrationEventBus(pool)
	expectedErr := errors.New("consumer exploded")
	sharedevent.Subscribe(
		eventBus,
		invoiceevents.EventNameInvoiceCreated,
		"billing",
		sharedevent.HandlerFunc(func(context.Context, sharedevent.Event) error {
			return expectedErr
		}),
	)

	err = sharedevent.Publish(
		ctx,
		eventBus,
		"invoices",
		invoiceevents.NewInvoiceCreated(
			ctx,
			"invoice-123",
			"customer-456",
			1599,
			time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC),
		),
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected publish error %v, got %v", expectedErr, err)
	}

	recordedEvents := listOutboxEvents(ctx, t, pool)
	if len(recordedEvents) != 1 {
		t.Fatalf("expected one failed outbox event, got %d", len(recordedEvents))
	}

	if recordedEvents[0].Status != "failed" {
		t.Fatalf("expected failed status, got %q", recordedEvents[0].Status)
	}

	if recordedEvents[0].ErrorMessage != expectedErr.Error() {
		t.Fatalf("expected error message %q, got %q", expectedErr.Error(), recordedEvents[0].ErrorMessage)
	}
}

func TestPhase6CustomerCreatedUsesStandardizedEnvelopeInOutbox(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer pool.Close()

	eventBus := newIntegrationEventBus(pool)
	customerRepository := customerpersistence.NewPostgresRepository(pool)
	createCustomer := customersusecases.NewCreateCustomer(customerRepository, eventBus)

	customer, err := createCustomer.Execute(ctx, customersusecases.CreateCustomerInput{
		Name:     "Atlas Co",
		Document: "33322211100",
		Email:    "contracts@atlas.io",
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	recordedEvent := outboxEventByName(ctx, t, pool, customerevents.EventNameCustomerCreated)
	if recordedEvent.Status != "processed" {
		t.Fatalf("expected processed outbox event, got %q", recordedEvent.Status)
	}

	var payload struct {
		Metadata struct {
			EventName   string `json:"event_name"`
			AggregateID string `json:"aggregate_id"`
		} `json:"metadata"`
		Payload struct {
			CustomerID string `json:"customer_id"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(recordedEvent.Payload, &payload); err != nil {
		t.Fatalf("decode customer created outbox payload: %v", err)
	}

	if payload.Metadata.EventName != customerevents.EventNameCustomerCreated {
		t.Fatalf("expected event name %q, got %q", customerevents.EventNameCustomerCreated, payload.Metadata.EventName)
	}

	if payload.Metadata.AggregateID != customer.ID {
		t.Fatalf("expected aggregate id %q, got %q", customer.ID, payload.Metadata.AggregateID)
	}

	if payload.Payload.CustomerID != customer.ID {
		t.Fatalf("expected customer id %q, got %q", customer.ID, payload.Payload.CustomerID)
	}
}

func TestPhase7DuplicateBillingRequestedDoesNotCreateDuplicatePayment(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer pool.Close()

	eventBus := newIntegrationEventBusWithFaultProfile(pool, config.FaultProfileDuplicateBillingRequest)
	customerModule := customers.NewModule(pool, eventBus)
	invoices.NewModule(pool, customerModule.ExistenceChecker(), eventBus)
	billingModule := billing.NewModule(pool, eventBus)
	_ = payments.NewModule(pool, billingModule.PaymentPort(), eventBus, newIntegrationGateway(config.FaultProfileDuplicateBillingRequest, time.Second), payments.ModuleConfig{
		GatewayTimeout: time.Second,
	})

	customerRepository := customerpersistence.NewPostgresRepository(pool)
	createCustomer := customersusecases.NewCreateCustomer(customerRepository, eventBus)
	customer, err := createCustomer.Execute(ctx, customersusecases.CreateCustomerInput{
		Name:     "Atlas Co",
		Document: "51022033011",
		Email:    "duplicate@atlas.io",
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	invoiceRepository := invoicepersistence.NewPostgresRepository(pool)
	createInvoice := invoicesusecases.NewCreateInvoice(invoiceRepository, customerModule.ExistenceChecker(), eventBus)
	invoice, err := createInvoice.Execute(ctx, invoicesusecases.CreateInvoiceInput{
		CustomerID:  customer.ID,
		AmountCents: 4500,
		DueDate:     "2026-03-31",
	})
	if err != nil {
		t.Fatalf("create invoice with duplicated delivery: %v", err)
	}

	paymentRepository := paymentpersistence.NewPostgresRepository(pool)
	storedPayments, err := paymentRepository.ListByInvoiceID(ctx, invoice.ID)
	if err != nil {
		t.Fatalf("list payments by invoice: %v", err)
	}

	if len(storedPayments) != 1 {
		t.Fatalf("expected one approved payment after duplicated BillingRequested, got %d", len(storedPayments))
	}

	if storedPayments[0].Status() != "Approved" {
		t.Fatalf("expected approved payment after duplicated BillingRequested, got %q", storedPayments[0].Status())
	}
}

func TestPhase7PaymentFlakyFirstAllowsSuccessfulManualRetry(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer pool.Close()

	eventBus := newIntegrationEventBusWithFaultProfile(pool, config.FaultProfilePaymentFlakyFirst)
	customerModule := customers.NewModule(pool, eventBus)
	invoices.NewModule(pool, customerModule.ExistenceChecker(), eventBus)
	billingModule := billing.NewModule(pool, eventBus)
	gateway := newIntegrationGateway(config.FaultProfilePaymentFlakyFirst, time.Second)
	_ = payments.NewModule(pool, billingModule.PaymentPort(), eventBus, gateway, payments.ModuleConfig{
		GatewayTimeout: time.Second,
	})

	customerRepository := customerpersistence.NewPostgresRepository(pool)
	createCustomer := customersusecases.NewCreateCustomer(customerRepository, eventBus)
	customer, err := createCustomer.Execute(ctx, customersusecases.CreateCustomerInput{
		Name:     "Atlas Co",
		Document: "51022033012",
		Email:    "flaky@atlas.io",
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	invoiceRepository := invoicepersistence.NewPostgresRepository(pool)
	createInvoice := invoicesusecases.NewCreateInvoice(invoiceRepository, customerModule.ExistenceChecker(), eventBus)
	invoice, err := createInvoice.Execute(ctx, invoicesusecases.CreateInvoiceInput{
		CustomerID:  customer.ID,
		AmountCents: 5500,
		DueDate:     "2026-03-31",
	})
	if err != nil {
		t.Fatalf("create invoice with flaky gateway: %v", err)
	}

	processBillingRequest := paymentsusecases.NewProcessBillingRequest(
		paymentpersistence.NewPostgresRepository(pool),
		gateway,
		sharedpostgres.NewTxManager(pool),
		eventBus,
		time.Second,
	)
	processPayment := paymentsusecases.NewProcessPayment(billingModule.PaymentPort(), processBillingRequest)

	payment, err := processPayment.Execute(ctx, paymentsusecases.ProcessPaymentInput{InvoiceID: invoice.ID})
	if err != nil {
		t.Fatalf("manual retry payment: %v", err)
	}

	if payment.Status != "Approved" {
		t.Fatalf("expected approved manual retry after flaky first call, got %q", payment.Status)
	}

	storedInvoice, err := invoiceRepository.GetByID(ctx, invoice.ID)
	if err != nil {
		t.Fatalf("get stored invoice: %v", err)
	}

	if storedInvoice.Status() != "Paid" {
		t.Fatalf("expected paid invoice after manual retry, got %q", storedInvoice.Status())
	}

	storedPayments, err := paymentpersistence.NewPostgresRepository(pool).ListByInvoiceID(ctx, invoice.ID)
	if err != nil {
		t.Fatalf("list stored payments: %v", err)
	}

	if len(storedPayments) != 2 {
		t.Fatalf("expected two payment attempts, got %d", len(storedPayments))
	}

	if storedPayments[0].FailureCategory() != entities.FailureCategoryGatewayError {
		t.Fatalf("expected first attempt gateway_error, got %q", storedPayments[0].FailureCategory())
	}

	if storedPayments[1].Status() != "Approved" {
		t.Fatalf("expected second attempt approved, got %q", storedPayments[1].Status())
	}
}

func TestPhase7InjectedConsumerFailureLeavesBillingRequestedAndOutboxFailed(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer pool.Close()

	eventBus := newIntegrationEventBusWithFaultProfile(pool, config.FaultProfileEventConsumerFailure)
	customerModule := customers.NewModule(pool, eventBus)
	invoices.NewModule(pool, customerModule.ExistenceChecker(), eventBus)
	billingModule := billing.NewModule(pool, eventBus)
	_ = payments.NewModule(pool, billingModule.PaymentPort(), eventBus, newIntegrationGateway(config.FaultProfileEventConsumerFailure, time.Second), payments.ModuleConfig{
		GatewayTimeout: time.Second,
	})

	customerRepository := customerpersistence.NewPostgresRepository(pool)
	createCustomer := customersusecases.NewCreateCustomer(customerRepository, eventBus)
	customer, err := createCustomer.Execute(ctx, customersusecases.CreateCustomerInput{
		Name:     "Atlas Co",
		Document: "51022033013",
		Email:    "consumer-failure@atlas.io",
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	invoiceRepository := invoicepersistence.NewPostgresRepository(pool)
	createInvoice := invoicesusecases.NewCreateInvoice(invoiceRepository, customerModule.ExistenceChecker(), eventBus)
	_, err = createInvoice.Execute(ctx, invoicesusecases.CreateInvoiceInput{
		CustomerID:  customer.ID,
		AmountCents: 6500,
		DueDate:     "2026-03-31",
	})
	if !errors.Is(err, sharedevent.ErrInjectedConsumerFailure) {
		t.Fatalf("expected injected consumer failure, got %v", err)
	}

	storedInvoices, getErr := invoiceRepository.ListByCustomerID(ctx, customer.ID)
	if getErr != nil {
		t.Fatalf("list stored invoices: %v", getErr)
	}

	if len(storedInvoices) != 1 {
		t.Fatalf("expected one stored invoice after consumer failure, got %d", len(storedInvoices))
	}

	if storedInvoices[0].Status() != "Pending" {
		t.Fatalf("expected pending invoice after consumer failure, got %q", storedInvoices[0].Status())
	}

	storedBilling, getErr := billingpersistence.NewPostgresRepository(pool).GetByInvoiceID(ctx, storedInvoices[0].ID())
	if getErr != nil {
		t.Fatalf("get stored billing: %v", getErr)
	}

	if storedBilling.Status() != "Requested" {
		t.Fatalf("expected requested billing after consumer failure, got %q", storedBilling.Status())
	}

	storedPayments, getErr := paymentpersistence.NewPostgresRepository(pool).ListByInvoiceID(ctx, storedInvoices[0].ID())
	if getErr != nil {
		t.Fatalf("list payments by invoice: %v", getErr)
	}

	if len(storedPayments) != 0 {
		t.Fatalf("expected no payment created after consumer failure, got %d", len(storedPayments))
	}

	billingRequested := outboxEventByName(ctx, t, pool, "BillingRequested")
	if billingRequested.Status != "failed" {
		t.Fatalf("expected BillingRequested outbox event failed, got %q", billingRequested.Status)
	}

	invoiceCreated := outboxEventByName(ctx, t, pool, invoiceevents.EventNameInvoiceCreated)
	if invoiceCreated.Status != "failed" {
		t.Fatalf("expected InvoiceCreated outbox event failed after downstream consumer failure, got %q", invoiceCreated.Status)
	}
}

func TestPhase7OutboxAppendFailureKeepsInvoicePersistedAndBlocksConsumers(t *testing.T) {
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
	customer, err := customerentities.NewCustomer(
		uuid.NewString(),
		"Atlas Co",
		"51022033014",
		"outbox-failure@atlas.io",
		time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("new customer: %v", err)
	}

	if err := customerRepository.Save(ctx, customer); err != nil {
		t.Fatalf("save customer: %v", err)
	}

	eventBus := newIntegrationEventBusWithFaultProfile(pool, config.FaultProfileOutboxAppendFailure)
	customerModule := customers.NewModule(pool, eventBus)
	invoiceRepository := invoicepersistence.NewPostgresRepository(pool)
	createInvoice := invoicesusecases.NewCreateInvoice(invoiceRepository, customerModule.ExistenceChecker(), eventBus)

	_, err = createInvoice.Execute(ctx, invoicesusecases.CreateInvoiceInput{
		CustomerID:  customer.ID(),
		AmountCents: 7500,
		DueDate:     "2026-03-31",
	})
	if !errors.Is(err, runtimefaults.ErrSimulatedOutboxAppendFailure) {
		t.Fatalf("expected simulated outbox append failure, got %v", err)
	}

	storedInvoices, getErr := invoiceRepository.ListByCustomerID(ctx, customer.ID())
	if getErr != nil {
		t.Fatalf("list stored invoices: %v", getErr)
	}

	if len(storedInvoices) != 1 {
		t.Fatalf("expected one persisted invoice after outbox append failure, got %d", len(storedInvoices))
	}

	if storedInvoices[0].Status() != "Pending" {
		t.Fatalf("expected persisted pending invoice after outbox append failure, got %q", storedInvoices[0].Status())
	}

	_, getErr = billingpersistence.NewPostgresRepository(pool).GetByInvoiceID(ctx, storedInvoices[0].ID())
	if !errors.Is(getErr, billingpublic.ErrBillingNotFound) {
		t.Fatalf("expected no billing after outbox append failure, got %v", getErr)
	}

	if count := countOutboxEvents(ctx, t, pool); count != 0 {
		t.Fatalf("expected zero outbox events after append failure, got %d", count)
	}
}

func newIntegrationEventBus(pool *pgxpool.Pool) *sharedevent.SyncBus {
	return sharedevent.NewSyncBus(outbox.NewPostgresRecorder(pool))
}

func newIntegrationEventBusWithFaultProfile(pool *pgxpool.Pool, profile config.FaultProfile) *sharedevent.SyncBus {
	recorder := runtimefaults.DecorateRecorder(profile, outbox.NewPostgresRecorder(pool))

	return sharedevent.NewSyncBusWithOptions(
		runtimefaults.EventBusOptions(profile, nil, recorder),
	)
}

func newIntegrationGateway(profile config.FaultProfile, timeout time.Duration) paymentports.PaymentGateway {
	return runtimefaults.DecorateGateway(profile, timeout, integration.NewMockGateway())
}

func countOutboxEvents(ctx context.Context, t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()

	var count int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox_events").Scan(&count); err != nil {
		t.Fatalf("count outbox events: %v", err)
	}

	return count
}

type outboxEventRow struct {
	EventName     string
	Status        string
	AggregateID   string
	CorrelationID string
	Payload       []byte
	ErrorMessage  string
}

func listOutboxEvents(ctx context.Context, t *testing.T, pool *pgxpool.Pool) []outboxEventRow {
	t.Helper()

	rows, err := pool.Query(ctx, `
		SELECT event_name, status, aggregate_id, correlation_id, payload, COALESCE(error_message, '')
		FROM outbox_events
		ORDER BY occurred_at, event_name
	`)
	if err != nil {
		t.Fatalf("list outbox events: %v", err)
	}
	defer rows.Close()

	var events []outboxEventRow
	for rows.Next() {
		var event outboxEventRow
		if err := rows.Scan(&event.EventName, &event.Status, &event.AggregateID, &event.CorrelationID, &event.Payload, &event.ErrorMessage); err != nil {
			t.Fatalf("scan outbox event: %v", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("iterate outbox events: %v", err)
	}

	return events
}

func outboxEventByName(ctx context.Context, t *testing.T, pool *pgxpool.Pool, eventName string) outboxEventRow {
	t.Helper()

	for _, event := range listOutboxEvents(ctx, t, pool) {
		if event.EventName == eventName {
			return event
		}
	}

	t.Fatalf("outbox event %s not found", eventName)
	return outboxEventRow{}
}
