package functional_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rgomids/atlas-erp-core/internal/billing"
	"github.com/rgomids/atlas-erp-core/internal/customers"
	"github.com/rgomids/atlas-erp-core/internal/invoices"
	"github.com/rgomids/atlas-erp-core/internal/payments"
	paymentports "github.com/rgomids/atlas-erp-core/internal/payments/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/payments/infrastructure/integration"
	"github.com/rgomids/atlas-erp-core/internal/shared/config"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	httpapi "github.com/rgomids/atlas-erp-core/internal/shared/http"
	"github.com/rgomids/atlas-erp-core/internal/shared/logging"
	"github.com/rgomids/atlas-erp-core/internal/shared/outbox"
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
	"github.com/rgomids/atlas-erp-core/internal/shared/runtimefaults"
	"github.com/rgomids/atlas-erp-core/test/support"
)

func TestPhase3HTTPFlowCompletesEndToEndViaInvoiceCreation(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer pool.Close()

	server := newFunctionalServer(t, pool, integration.NewMockGateway(), &bytes.Buffer{})
	defer server.Close()

	customerResponse := postJSON(t, server.URL+"/customers", `{"name":"Atlas Co","document":"12345678900","email":"team@atlas.io"}`)
	if customerResponse.StatusCode != http.StatusCreated {
		t.Fatalf("expected customer creation status 201, got %d", customerResponse.StatusCode)
	}
	var customer struct {
		ID string `json:"id"`
	}
	decodeResponse(t, customerResponse, &customer)

	invoiceResponse := postJSON(t, server.URL+"/invoices", `{"customer_id":"`+customer.ID+`","amount_cents":1599,"due_date":"2026-03-25"}`)
	if invoiceResponse.StatusCode != http.StatusCreated {
		t.Fatalf("expected invoice creation status 201, got %d", invoiceResponse.StatusCode)
	}
	var invoice struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	decodeResponse(t, invoiceResponse, &invoice)

	if invoice.Status != "Pending" {
		t.Fatalf("expected create invoice response to remain pending, got %q", invoice.Status)
	}

	listResponse, err := server.Client().Get(server.URL + "/customers/" + customer.ID + "/invoices")
	if err != nil {
		t.Fatalf("list customer invoices: %v", err)
	}
	defer listResponse.Body.Close()

	if listResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected list invoices status 200, got %d", listResponse.StatusCode)
	}

	var invoicesPayload struct {
		Items []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := json.NewDecoder(listResponse.Body).Decode(&invoicesPayload); err != nil {
		t.Fatalf("decode invoices payload: %v", err)
	}

	if len(invoicesPayload.Items) != 1 {
		t.Fatalf("expected 1 invoice, got %d", len(invoicesPayload.Items))
	}

	if invoicesPayload.Items[0].Status != "Paid" {
		t.Fatalf("expected paid invoice after automatic flow, got %q", invoicesPayload.Items[0].Status)
	}
}

func TestPhase3HTTPRetryAfterAutomaticFailure(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer pool.Close()

	failedServer := newFunctionalServer(t, pool, integration.NewMockGatewayWithStatus("Failed"), &bytes.Buffer{})
	defer failedServer.Close()

	customerResponse := postJSON(t, failedServer.URL+"/customers", `{"name":"Atlas Co","document":"12345678900","email":"team@atlas.io"}`)
	var customer struct {
		ID string `json:"id"`
	}
	decodeResponse(t, customerResponse, &customer)

	invoiceResponse := postJSON(t, failedServer.URL+"/invoices", `{"customer_id":"`+customer.ID+`","amount_cents":1599,"due_date":"2026-03-25"}`)
	if invoiceResponse.StatusCode != http.StatusCreated {
		t.Fatalf("expected invoice creation status 201, got %d", invoiceResponse.StatusCode)
	}
	var invoice struct {
		ID string `json:"id"`
	}
	decodeResponse(t, invoiceResponse, &invoice)

	listResponse, err := failedServer.Client().Get(failedServer.URL + "/customers/" + customer.ID + "/invoices")
	if err != nil {
		t.Fatalf("list invoices after failed payment: %v", err)
	}
	defer listResponse.Body.Close()

	var beforeRetry struct {
		Items []struct {
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := json.NewDecoder(listResponse.Body).Decode(&beforeRetry); err != nil {
		t.Fatalf("decode invoices before retry: %v", err)
	}

	if beforeRetry.Items[0].Status != "Pending" {
		t.Fatalf("expected pending invoice after failed automatic payment, got %q", beforeRetry.Items[0].Status)
	}

	approvedServer := newFunctionalServer(t, pool, integration.NewMockGateway(), &bytes.Buffer{})
	defer approvedServer.Close()

	paymentResponse := postJSON(t, approvedServer.URL+"/payments", `{"invoice_id":"`+invoice.ID+`"}`)
	if paymentResponse.StatusCode != http.StatusCreated {
		t.Fatalf("expected manual retry status 201, got %d", paymentResponse.StatusCode)
	}

	var payment struct {
		Status string `json:"status"`
	}
	decodeResponse(t, paymentResponse, &payment)
	if payment.Status != "Approved" {
		t.Fatalf("expected approved retry payment, got %q", payment.Status)
	}

	finalListResponse, err := approvedServer.Client().Get(approvedServer.URL + "/customers/" + customer.ID + "/invoices")
	if err != nil {
		t.Fatalf("list invoices after retry: %v", err)
	}
	defer finalListResponse.Body.Close()

	var afterRetry struct {
		Items []struct {
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := json.NewDecoder(finalListResponse.Body).Decode(&afterRetry); err != nil {
		t.Fatalf("decode invoices after retry: %v", err)
	}

	if afterRetry.Items[0].Status != "Paid" {
		t.Fatalf("expected paid invoice after retry, got %q", afterRetry.Items[0].Status)
	}
}

func TestPhase3HTTPInvalidInputReturnsCanonicalErrorAndTraceability(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer pool.Close()

	logBuffer := &bytes.Buffer{}
	server := newFunctionalServer(t, pool, integration.NewMockGateway(), logBuffer)
	defer server.Close()

	response := postJSONWithRequestID(t, server.URL+"/customers", `{"name":"Atlas Co","email":"team@atlas.io"}`, "req-functional-002")
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected customer validation status 400, got %d", response.StatusCode)
	}

	if response.Header.Get("X-Correlation-ID") != "req-functional-002" {
		t.Fatalf("expected correlation id header to be preserved, got %q", response.Header.Get("X-Correlation-ID"))
	}

	var errorPayload httpapi.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&errorPayload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if errorPayload.Error != "invalid_input" {
		t.Fatalf("expected invalid_input error, got %q", errorPayload.Error)
	}

	if errorPayload.Message != "document is required" {
		t.Fatalf("expected document required message, got %q", errorPayload.Message)
	}

	if errorPayload.RequestID != "req-functional-002" {
		t.Fatalf("expected request_id to match correlation id, got %q", errorPayload.RequestID)
	}

	logOutput := logBuffer.String()
	for _, fragment := range []string{`"module":"customers"`, `"request_id":"req-functional-002"`} {
		if !strings.Contains(logOutput, fragment) {
			t.Fatalf("expected log output to contain %s, got %s", fragment, logOutput)
		}
	}
}

func TestPhase3HTTPLogsEventsWithEmitterAndConsumerModules(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer pool.Close()

	logBuffer := &bytes.Buffer{}
	server := newFunctionalServer(t, pool, integration.NewMockGateway(), logBuffer)
	defer server.Close()

	customerResponse := postJSONWithRequestID(t, server.URL+"/customers", `{"name":"Atlas Co","document":"12345678900","email":"team@atlas.io"}`, "req-functional-003")
	var customer struct {
		ID string `json:"id"`
	}
	decodeResponse(t, customerResponse, &customer)

	invoiceResponse := postJSONWithRequestID(t, server.URL+"/invoices", `{"customer_id":"`+customer.ID+`","amount_cents":1599,"due_date":"2026-03-25"}`, "req-functional-003")
	defer invoiceResponse.Body.Close()

	if invoiceResponse.StatusCode != http.StatusCreated {
		t.Fatalf("expected invoice creation status 201, got %d", invoiceResponse.StatusCode)
	}

	logOutput := logBuffer.String()
	for _, fragment := range []string{
		`"event":"InvoiceCreated"`,
		`"event":"BillingRequested"`,
		`"event":"PaymentApproved"`,
		`"emitter_module":"invoices"`,
		`"consumer_module":"billing"`,
		`"consumer_module":"payments"`,
		`"invoice_id":"`,
		`"customer_id":"`,
		`"attempt_number":1`,
		`"request_id":"req-functional-003"`,
	} {
		if !strings.Contains(logOutput, fragment) {
			t.Fatalf("expected log output to contain %s, got %s", fragment, logOutput)
		}
	}
}

func TestPhase4HTTPAutomaticTimeoutKeepsInvoicePendingAndReturnsCreatedInvoice(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer pool.Close()

	server := newFunctionalServerWithTimeout(t, pool, integration.NewMockGatewayWithDelay("Approved", 25*time.Millisecond), &bytes.Buffer{}, 5*time.Millisecond)
	defer server.Close()

	customerResponse := postJSON(t, server.URL+"/customers", `{"name":"Atlas Co","document":"12345678900","email":"team@atlas.io"}`)
	var customer struct {
		ID string `json:"id"`
	}
	decodeResponse(t, customerResponse, &customer)

	invoiceResponse := postJSON(t, server.URL+"/invoices", `{"customer_id":"`+customer.ID+`","amount_cents":1599,"due_date":"2026-03-25"}`)
	if invoiceResponse.StatusCode != http.StatusCreated {
		t.Fatalf("expected invoice creation status 201, got %d", invoiceResponse.StatusCode)
	}

	var invoice struct {
		ID string `json:"id"`
	}
	decodeResponse(t, invoiceResponse, &invoice)

	listResponse, err := server.Client().Get(server.URL + "/customers/" + customer.ID + "/invoices")
	if err != nil {
		t.Fatalf("list invoices after timeout: %v", err)
	}
	defer listResponse.Body.Close()

	var invoicesPayload struct {
		Items []struct {
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := json.NewDecoder(listResponse.Body).Decode(&invoicesPayload); err != nil {
		t.Fatalf("decode invoices after timeout: %v", err)
	}

	if invoicesPayload.Items[0].Status != "Pending" {
		t.Fatalf("expected pending invoice after gateway timeout, got %q", invoicesPayload.Items[0].Status)
	}
}

func TestPhase4HTTPManualRetryReturnsFailedPayloadOnTechnicalFailure(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer pool.Close()

	failedServer := newFunctionalServer(t, pool, integration.NewMockGatewayWithStatus("Failed"), &bytes.Buffer{})
	defer failedServer.Close()

	customerResponse := postJSON(t, failedServer.URL+"/customers", `{"name":"Atlas Co","document":"12345678900","email":"team@atlas.io"}`)
	var customer struct {
		ID string `json:"id"`
	}
	decodeResponse(t, customerResponse, &customer)

	invoiceResponse := postJSON(t, failedServer.URL+"/invoices", `{"customer_id":"`+customer.ID+`","amount_cents":1599,"due_date":"2026-03-25"}`)
	var invoice struct {
		ID string `json:"id"`
	}
	decodeResponse(t, invoiceResponse, &invoice)

	timeoutServer := newFunctionalServerWithTimeout(t, pool, integration.NewMockGatewayWithDelay("Approved", 25*time.Millisecond), &bytes.Buffer{}, 5*time.Millisecond)
	defer timeoutServer.Close()

	paymentResponse := postJSON(t, timeoutServer.URL+"/payments", `{"invoice_id":"`+invoice.ID+`"}`)
	if paymentResponse.StatusCode != http.StatusCreated {
		t.Fatalf("expected manual retry status 201, got %d", paymentResponse.StatusCode)
	}

	var payment struct {
		Status          string `json:"status"`
		FailureCategory string `json:"failure_category"`
		AttemptNumber   int    `json:"attempt_number"`
	}
	decodeResponse(t, paymentResponse, &payment)

	if payment.Status != "Failed" {
		t.Fatalf("expected failed payment, got %q", payment.Status)
	}

	if payment.FailureCategory != "gateway_timeout" {
		t.Fatalf("expected gateway_timeout failure category, got %q", payment.FailureCategory)
	}

	if payment.AttemptNumber != 2 {
		t.Fatalf("expected retry attempt number 2, got %d", payment.AttemptNumber)
	}
}

func TestPhase7HTTPPaymentTimeoutProfileKeepsInvoicePending(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer pool.Close()

	server := newFunctionalServerWithProfile(t, pool, config.FaultProfilePaymentTimeout, &bytes.Buffer{}, 5*time.Millisecond)
	defer server.Close()

	customerResponse := postJSON(t, server.URL+"/customers", `{"name":"Atlas Co","document":"12345678900","email":"team@atlas.io"}`)
	var customer struct {
		ID string `json:"id"`
	}
	decodeResponse(t, customerResponse, &customer)

	invoiceResponse := postJSON(t, server.URL+"/invoices", `{"customer_id":"`+customer.ID+`","amount_cents":1599,"due_date":"2026-03-25"}`)
	if invoiceResponse.StatusCode != http.StatusCreated {
		t.Fatalf("expected invoice creation status 201, got %d", invoiceResponse.StatusCode)
	}

	listResponse, err := server.Client().Get(server.URL + "/customers/" + customer.ID + "/invoices")
	if err != nil {
		t.Fatalf("list invoices after profile timeout: %v", err)
	}
	defer listResponse.Body.Close()

	var payload struct {
		Items []struct {
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := json.NewDecoder(listResponse.Body).Decode(&payload); err != nil {
		t.Fatalf("decode invoices payload: %v", err)
	}

	if payload.Items[0].Status != "Pending" {
		t.Fatalf("expected pending invoice after payment_timeout profile, got %q", payload.Items[0].Status)
	}
}

func TestPhase7HTTPRepeatedPaymentsRetryStopsAfterSuccess(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer pool.Close()

	server := newFunctionalServerWithProfile(t, pool, config.FaultProfilePaymentFlakyFirst, &bytes.Buffer{}, time.Second)
	defer server.Close()

	customerResponse := postJSON(t, server.URL+"/customers", `{"name":"Atlas Co","document":"12345678900","email":"team@atlas.io"}`)
	var customer struct {
		ID string `json:"id"`
	}
	decodeResponse(t, customerResponse, &customer)

	invoiceResponse := postJSON(t, server.URL+"/invoices", `{"customer_id":"`+customer.ID+`","amount_cents":1599,"due_date":"2026-03-25"}`)
	var invoice struct {
		ID string `json:"id"`
	}
	decodeResponse(t, invoiceResponse, &invoice)

	firstRetry := postJSON(t, server.URL+"/payments", `{"invoice_id":"`+invoice.ID+`"}`)
	if firstRetry.StatusCode != http.StatusCreated {
		t.Fatalf("expected first retry status 201, got %d", firstRetry.StatusCode)
	}

	var approved struct {
		Status        string `json:"status"`
		AttemptNumber int    `json:"attempt_number"`
	}
	decodeResponse(t, firstRetry, &approved)

	if approved.Status != "Approved" || approved.AttemptNumber != 2 {
		t.Fatalf("expected approved second attempt, got status=%q attempt=%d", approved.Status, approved.AttemptNumber)
	}

	secondRetry := postJSON(t, server.URL+"/payments", `{"invoice_id":"`+invoice.ID+`"}`)
	defer secondRetry.Body.Close()

	if secondRetry.StatusCode != http.StatusConflict {
		t.Fatalf("expected second retry conflict status 409, got %d", secondRetry.StatusCode)
	}

	var conflict httpapi.ErrorResponse
	if err := json.NewDecoder(secondRetry.Body).Decode(&conflict); err != nil {
		t.Fatalf("decode conflict payload: %v", err)
	}

	if conflict.Error != "payment_conflict" {
		t.Fatalf("expected payment_conflict error, got %q", conflict.Error)
	}
}

func TestPhase7HTTPLogsInjectedConsumerFailure(t *testing.T) {
	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer pool.Close()

	logBuffer := &bytes.Buffer{}
	server := newFunctionalServerWithProfile(t, pool, config.FaultProfileEventConsumerFailure, logBuffer, time.Second)
	defer server.Close()

	customerResponse := postJSONWithRequestID(t, server.URL+"/customers", `{"name":"Atlas Co","document":"12345678900","email":"team@atlas.io"}`, "req-functional-phase7-001")
	var customer struct {
		ID string `json:"id"`
	}
	decodeResponse(t, customerResponse, &customer)

	response := postJSONWithRequestID(t, server.URL+"/invoices", `{"customer_id":"`+customer.ID+`","amount_cents":1599,"due_date":"2026-03-25"}`, "req-functional-phase7-001")
	defer response.Body.Close()

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected internal error status 500, got %d", response.StatusCode)
	}

	logOutput := logBuffer.String()
	for _, fragment := range []string{
		`"event":"BillingRequested"`,
		`"consumer_module":"payments"`,
		`"error_type":"infrastructure_error"`,
		`"request_id":"req-functional-phase7-001"`,
	} {
		if !strings.Contains(logOutput, fragment) {
			t.Fatalf("expected log output to contain %s, got %s", fragment, logOutput)
		}
	}
}

func newFunctionalServer(t *testing.T, pool *pgxpool.Pool, gateway paymentports.PaymentGateway, logWriter *bytes.Buffer) *httptest.Server {
	return newFunctionalServerWithTimeout(t, pool, gateway, logWriter, time.Second)
}

func newFunctionalServerWithTimeout(t *testing.T, pool *pgxpool.Pool, gateway paymentports.PaymentGateway, logWriter *bytes.Buffer, timeout time.Duration) *httptest.Server {
	t.Helper()

	logger, err := logging.NewWithWriter("info", logWriter)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	eventBus := sharedevent.NewSyncBus(outbox.NewPostgresRecorder(pool))
	customerModule := customers.NewModule(pool, eventBus)
	invoiceModule := invoices.NewModule(pool, customerModule.ExistenceChecker(), eventBus)
	billingModule := billing.NewModule(pool, eventBus)
	paymentModule := payments.NewModule(pool, billingModule.PaymentPort(), eventBus, gateway, payments.ModuleConfig{
		GatewayTimeout: timeout,
	})

	return httptest.NewServer(httpapi.NewRouter(
		logger,
		"X-Correlation-ID",
		customerModule.Routes,
		invoiceModule.Routes,
		paymentModule.Routes,
	))
}

func newFunctionalServerWithProfile(
	t *testing.T,
	pool *pgxpool.Pool,
	profile config.FaultProfile,
	logWriter *bytes.Buffer,
	timeout time.Duration,
) *httptest.Server {
	t.Helper()

	logger, err := logging.NewWithWriter("info", logWriter)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	recorder := runtimefaults.DecorateRecorder(profile, outbox.NewPostgresRecorder(pool))
	eventBus := sharedevent.NewSyncBusWithOptions(runtimefaults.EventBusOptions(profile, nil, recorder))
	customerModule := customers.NewModule(pool, eventBus)
	invoiceModule := invoices.NewModule(pool, customerModule.ExistenceChecker(), eventBus)
	billingModule := billing.NewModule(pool, eventBus)
	gateway := runtimefaults.DecorateGateway(profile, timeout, integration.NewMockGateway())
	paymentModule := payments.NewModule(pool, billingModule.PaymentPort(), eventBus, gateway, payments.ModuleConfig{
		GatewayTimeout: timeout,
	})

	return httptest.NewServer(httpapi.NewRouter(
		logger,
		"X-Correlation-ID",
		customerModule.Routes,
		invoiceModule.Routes,
		paymentModule.Routes,
	))
}

func postJSON(t *testing.T, url string, payload string) *http.Response {
	t.Helper()

	return postJSONWithRequestID(t, url, payload, "")
}

func postJSONWithRequestID(t *testing.T, url string, payload string, requestID string) *http.Response {
	t.Helper()

	request, err := http.NewRequest(http.MethodPost, url, strings.NewReader(payload))
	if err != nil {
		t.Fatalf("create post request %s: %v", url, err)
	}

	request.Header.Set("Content-Type", "application/json")
	if requestID != "" {
		request.Header.Set("X-Correlation-ID", requestID)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post %s: %v", url, err)
	}

	return response
}

func decodeResponse(t *testing.T, response *http.Response, target any) {
	t.Helper()
	defer response.Body.Close()

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
