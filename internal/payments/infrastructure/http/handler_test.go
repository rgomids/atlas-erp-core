package paymentshttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	billingports "github.com/rgomids/atlas-erp-core/internal/billing/application/ports"
	billingentities "github.com/rgomids/atlas-erp-core/internal/billing/domain/entities"
	paymentports "github.com/rgomids/atlas-erp-core/internal/payments/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/payments/application/usecases"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/repositories"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	httpapi "github.com/rgomids/atlas-erp-core/internal/shared/http"
	"github.com/rgomids/atlas-erp-core/internal/shared/logging"
)

type paymentRepositoryStub struct {
	byID             map[string]entities.Payment
	byInvoice        map[string][]string
	byBillingAttempt map[string]string
}

var _ repositories.PaymentRepository = (*paymentRepositoryStub)(nil)

func newPaymentRepositoryStub() *paymentRepositoryStub {
	return &paymentRepositoryStub{
		byID:             map[string]entities.Payment{},
		byInvoice:        map[string][]string{},
		byBillingAttempt: map[string]string{},
	}
}

func (repository *paymentRepositoryStub) HasApprovedByInvoiceID(_ context.Context, invoiceID string) (bool, error) {
	for _, paymentID := range repository.byInvoice[invoiceID] {
		if repository.byID[paymentID].Status() == entities.StatusApproved {
			return true, nil
		}
	}

	return false, nil
}

func (repository *paymentRepositoryStub) Save(_ context.Context, payment entities.Payment) error {
	if _, exists := repository.byBillingAttempt[paymentAttemptKey(payment.BillingID(), payment.AttemptNumber())]; exists {
		return entities.ErrPaymentAlreadyExists
	}

	repository.store(payment)
	return nil
}

func (repository *paymentRepositoryStub) Update(_ context.Context, payment entities.Payment) error {
	if _, exists := repository.byID[payment.ID()]; !exists {
		return entities.ErrPaymentNotFound
	}

	repository.store(payment)
	return nil
}

func (repository *paymentRepositoryStub) GetByID(_ context.Context, paymentID string) (entities.Payment, error) {
	payment, ok := repository.byID[paymentID]
	if !ok {
		return entities.Payment{}, entities.ErrPaymentNotFound
	}

	return payment, nil
}

func (repository *paymentRepositoryStub) GetByBillingIDAndAttempt(_ context.Context, billingID string, attemptNumber int) (entities.Payment, error) {
	paymentID, ok := repository.byBillingAttempt[paymentAttemptKey(billingID, attemptNumber)]
	if !ok {
		return entities.Payment{}, entities.ErrPaymentNotFound
	}

	return repository.byID[paymentID], nil
}

func (repository *paymentRepositoryStub) ListByInvoiceID(_ context.Context, invoiceID string) ([]entities.Payment, error) {
	var payments []entities.Payment
	for _, paymentID := range repository.byInvoice[invoiceID] {
		payments = append(payments, repository.byID[paymentID])
	}

	return payments, nil
}

func (repository *paymentRepositoryStub) store(payment entities.Payment) {
	repository.byID[payment.ID()] = payment
	repository.byBillingAttempt[paymentAttemptKey(payment.BillingID(), payment.AttemptNumber())] = payment.ID()

	for _, existingID := range repository.byInvoice[payment.InvoiceID()] {
		if existingID == payment.ID() {
			return
		}
	}

	repository.byInvoice[payment.InvoiceID()] = append(repository.byInvoice[payment.InvoiceID()], payment.ID())
}

func paymentAttemptKey(billingID string, attemptNumber int) string {
	return fmt.Sprintf("%s#%d", billingID, attemptNumber)
}

type billingPortStub struct {
	snapshot billingports.BillingSnapshot
	err      error
}

func (port *billingPortStub) GetProcessableBillingByInvoiceID(context.Context, string) (billingports.BillingSnapshot, error) {
	if port.err != nil {
		return billingports.BillingSnapshot{}, port.err
	}

	return port.snapshot, nil
}

type paymentGatewayStub struct {
	result paymentports.GatewayResult
	err    error
}

func (gateway paymentGatewayStub) Process(context.Context, paymentports.GatewayRequest) (paymentports.GatewayResult, error) {
	return gateway.result, gateway.err
}

type txManagerStub struct{}

func (txManagerStub) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestCreatePaymentReturnsCreatedPayload(t *testing.T) {
	t.Parallel()

	server := newPaymentTestServer(t, newPaymentRepositoryStub(), &billingPortStub{
		snapshot: billingports.BillingSnapshot{
			ID:            "a4a40fd7-50ac-43c6-b6d1-a98ee0952603",
			InvoiceID:     "1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe",
			CustomerID:    "7adf3d42-7b1d-4d2b-a7d6-5d977b7576aa",
			AmountCents:   1599,
			Status:        "Requested",
			DueDate:       time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
			AttemptNumber: 1,
		},
	}, paymentGatewayStub{result: paymentports.GatewayResult{Status: "Approved", GatewayReference: "gw-001"}})
	defer server.Close()

	response := performPaymentRequest(t, server, http.MethodPost, "/payments", `{"invoice_id":"1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe"}`, "req-payment-001")
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.StatusCode)
	}

	var payload struct {
		ID            string `json:"id"`
		Status        string `json:"status"`
		AttemptNumber int    `json:"attempt_number"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode payment payload: %v", err)
	}

	if payload.ID == "" {
		t.Fatal("expected payment id to be generated")
	}

	if payload.Status != "Approved" {
		t.Fatalf("expected approved payment, got %q", payload.Status)
	}

	if payload.AttemptNumber != 1 {
		t.Fatalf("expected attempt number 1, got %d", payload.AttemptNumber)
	}
}

func TestCreatePaymentReturnsFailedPayloadOnGatewayTechnicalFailure(t *testing.T) {
	t.Parallel()

	server := newPaymentTestServer(t, newPaymentRepositoryStub(), &billingPortStub{
		snapshot: billingports.BillingSnapshot{
			ID:            "a4a40fd7-50ac-43c6-b6d1-a98ee0952603",
			InvoiceID:     "1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe",
			CustomerID:    "7adf3d42-7b1d-4d2b-a7d6-5d977b7576aa",
			AmountCents:   1599,
			Status:        "Requested",
			DueDate:       time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
			AttemptNumber: 1,
		},
	}, paymentGatewayStub{err: context.DeadlineExceeded})
	defer server.Close()

	response := performPaymentRequest(t, server, http.MethodPost, "/payments", `{"invoice_id":"1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe"}`, "req-payment-technical")
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.StatusCode)
	}

	var payload struct {
		Status          string `json:"status"`
		FailureCategory string `json:"failure_category"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode failed payment payload: %v", err)
	}

	if payload.Status != "Failed" {
		t.Fatalf("expected failed payment, got %q", payload.Status)
	}

	if payload.FailureCategory != "gateway_timeout" {
		t.Fatalf("expected gateway_timeout failure category, got %q", payload.FailureCategory)
	}
}

func TestCreatePaymentRejectsMissingInvoiceIDAtHTTPBoundary(t *testing.T) {
	t.Parallel()

	server := newPaymentTestServer(t, newPaymentRepositoryStub(), &billingPortStub{}, paymentGatewayStub{})
	defer server.Close()

	response := performPaymentRequest(t, server, http.MethodPost, "/payments", `{}`, "req-payment-002")
	defer response.Body.Close()

	assertPaymentErrorResponse(t, response, http.StatusBadRequest, "invalid_input", "invoice_id is required", "req-payment-002")
}

func TestCreatePaymentMapsConflictWhenApprovedPaymentAlreadyExists(t *testing.T) {
	t.Parallel()

	repository := newPaymentRepositoryStub()
	approvedPayment, err := entities.RehydratePayment(
		"payment-001",
		"a4a40fd7-50ac-43c6-b6d1-a98ee0952603",
		"1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe",
		1,
		"billing:a4a40fd7-50ac-43c6-b6d1-a98ee0952603:attempt:1",
		"Approved",
		"gw-001",
		"",
		time.Now().Add(-time.Minute),
		time.Now(),
	)
	if err != nil {
		t.Fatalf("rehydrate payment: %v", err)
	}
	if err := repository.Save(context.Background(), approvedPayment); err != nil {
		t.Fatalf("seed approved payment: %v", err)
	}

	server := newPaymentTestServer(t, repository, &billingPortStub{
		snapshot: billingports.BillingSnapshot{
			ID:            "a4a40fd7-50ac-43c6-b6d1-a98ee0952603",
			InvoiceID:     "1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe",
			CustomerID:    "7adf3d42-7b1d-4d2b-a7d6-5d977b7576aa",
			AmountCents:   1599,
			Status:        "Requested",
			DueDate:       time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
			AttemptNumber: 2,
		},
	}, paymentGatewayStub{result: paymentports.GatewayResult{Status: "Approved", GatewayReference: "gw-001"}})
	defer server.Close()

	response := performPaymentRequest(t, server, http.MethodPost, "/payments", `{"invoice_id":"1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe"}`, "req-payment-003")
	defer response.Body.Close()

	assertPaymentErrorResponse(t, response, http.StatusConflict, "payment_conflict", "payment already exists for invoice", "req-payment-003")
}

func TestCreatePaymentMapsBillingNotFound(t *testing.T) {
	t.Parallel()

	server := newPaymentTestServer(t, newPaymentRepositoryStub(), &billingPortStub{err: billingentities.ErrBillingNotFound}, paymentGatewayStub{})
	defer server.Close()

	response := performPaymentRequest(t, server, http.MethodPost, "/payments", `{"invoice_id":"1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe"}`, "req-payment-004")
	defer response.Body.Close()

	assertPaymentErrorResponse(t, response, http.StatusNotFound, "billing_not_found", "billing not found", "req-payment-004")
}

func newPaymentTestServer(t *testing.T, repository *paymentRepositoryStub, billingPort *billingPortStub, gateway paymentGatewayStub) *httptest.Server {
	t.Helper()

	logger, err := logging.NewWithWriter("info", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	processBillingRequest := usecases.NewProcessBillingRequest(
		repository,
		gateway,
		txManagerStub{},
		sharedevent.NewSyncBus(),
		time.Second,
	)

	handler := NewHandler(
		usecases.NewProcessPayment(
			billingPort,
			processBillingRequest,
		),
	)

	return httptest.NewServer(httpapi.NewRouter(logger, "X-Correlation-ID", handler.Routes))
}

func performPaymentRequest(t *testing.T, server *httptest.Server, method, path, payload, requestID string) *http.Response {
	t.Helper()

	request, err := http.NewRequest(method, server.URL+path, strings.NewReader(payload))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Correlation-ID", requestID)

	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}

	return response
}

func assertPaymentErrorResponse(t *testing.T, response *http.Response, expectedStatus int, expectedError string, expectedMessage string, expectedRequestID string) {
	t.Helper()

	if response.StatusCode != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, response.StatusCode)
	}

	if response.Header.Get("X-Correlation-ID") != expectedRequestID {
		t.Fatalf("expected response header request id %q, got %q", expectedRequestID, response.Header.Get("X-Correlation-ID"))
	}

	var payload httpapi.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}

	if payload.Error != expectedError {
		t.Fatalf("expected error code %q, got %q", expectedError, payload.Error)
	}

	if payload.Message != expectedMessage {
		t.Fatalf("expected message %q, got %q", expectedMessage, payload.Message)
	}

	if payload.RequestID != expectedRequestID {
		t.Fatalf("expected request_id %q, got %q", expectedRequestID, payload.RequestID)
	}
}
