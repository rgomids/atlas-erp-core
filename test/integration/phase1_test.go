package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rgomids/atlas-erp-core/internal/billing"
	billingpersistence "github.com/rgomids/atlas-erp-core/internal/billing/infrastructure/persistence"
	"github.com/rgomids/atlas-erp-core/internal/customers"
	customersusecases "github.com/rgomids/atlas-erp-core/internal/customers/application/usecases"
	customerpersistence "github.com/rgomids/atlas-erp-core/internal/customers/infrastructure/persistence"
	customerevents "github.com/rgomids/atlas-erp-core/internal/customers/public/events"
	"github.com/rgomids/atlas-erp-core/internal/invoices"
	invoicesusecases "github.com/rgomids/atlas-erp-core/internal/invoices/application/usecases"
	invoicepersistence "github.com/rgomids/atlas-erp-core/internal/invoices/infrastructure/persistence"
	invoiceevents "github.com/rgomids/atlas-erp-core/internal/invoices/public/events"
	"github.com/rgomids/atlas-erp-core/internal/payments"
	paymentsusecases "github.com/rgomids/atlas-erp-core/internal/payments/application/usecases"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/integration"
	paymentpersistence "github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/persistence"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	"github.com/rgomids/atlas-erp-core/internal/shared/outbox"
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

func newIntegrationEventBus(pool *pgxpool.Pool) *sharedevent.SyncBus {
	return sharedevent.NewSyncBus(outbox.NewPostgresRecorder(pool))
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
