package paymentshttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rgomids/atlas-erp-core/internal/invoices/application/ports"
	invoiceentities "github.com/rgomids/atlas-erp-core/internal/invoices/domain/entities"
	paymentports "github.com/rgomids/atlas-erp-core/internal/payments/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/payments/application/usecases"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/repositories"
	httpapi "github.com/rgomids/atlas-erp-core/internal/shared/http"
	"github.com/rgomids/atlas-erp-core/internal/shared/logging"
)

type paymentRepositoryStub struct {
	byID      map[string]entities.Payment
	byInvoice map[string]string
}

var _ repositories.PaymentRepository = (*paymentRepositoryStub)(nil)

func newPaymentRepositoryStub() *paymentRepositoryStub {
	return &paymentRepositoryStub{
		byID:      map[string]entities.Payment{},
		byInvoice: map[string]string{},
	}
}

func (repository *paymentRepositoryStub) ExistsByInvoiceID(_ context.Context, invoiceID string) (bool, error) {
	_, exists := repository.byInvoice[invoiceID]
	return exists, nil
}

func (repository *paymentRepositoryStub) Save(_ context.Context, payment entities.Payment) error {
	repository.byID[payment.ID()] = payment
	repository.byInvoice[payment.InvoiceID()] = payment.ID()
	return nil
}

func (repository *paymentRepositoryStub) GetByID(_ context.Context, paymentID string) (entities.Payment, error) {
	payment, ok := repository.byID[paymentID]
	if !ok {
		return entities.Payment{}, entities.ErrInvalidPaymentID
	}

	return payment, nil
}

type invoicePaymentPortStub struct {
	snapshot ports.InvoiceSnapshot
	getErr   error
}

func (port *invoicePaymentPortStub) GetPayableInvoice(context.Context, string) (ports.InvoiceSnapshot, error) {
	if port.getErr != nil {
		return ports.InvoiceSnapshot{}, port.getErr
	}

	return port.snapshot, nil
}

func (port *invoicePaymentPortStub) MarkAsPaid(context.Context, string, time.Time) error {
	return nil
}

type paymentGatewayStub struct {
	result paymentports.GatewayResult
}

func (gateway paymentGatewayStub) Process(context.Context, paymentports.GatewayRequest) (paymentports.GatewayResult, error) {
	return gateway.result, nil
}

type txManagerStub struct{}

func (txManagerStub) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestCreatePaymentReturnsCreatedPayload(t *testing.T) {
	t.Parallel()

	server := newPaymentTestServer(t, newPaymentRepositoryStub(), &invoicePaymentPortStub{
		snapshot: ports.InvoiceSnapshot{
			ID:          "1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe",
			CustomerID:  "47df535a-56f3-473d-8f96-3c786bc4c537",
			AmountCents: 1599,
			Status:      "Pending",
			DueDate:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		},
	}, paymentGatewayStub{result: paymentports.GatewayResult{Status: "Approved", GatewayReference: "gw-001"}})
	defer server.Close()

	response := performPaymentRequest(t, server, http.MethodPost, "/payments", `{"invoice_id":"1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe"}`, "req-payment-001")
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.StatusCode)
	}

	var payload struct {
		ID     string `json:"id"`
		Status string `json:"status"`
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
}

func TestCreatePaymentRejectsMissingInvoiceIDAtHTTPBoundary(t *testing.T) {
	t.Parallel()

	server := newPaymentTestServer(t, newPaymentRepositoryStub(), &invoicePaymentPortStub{}, paymentGatewayStub{})
	defer server.Close()

	response := performPaymentRequest(t, server, http.MethodPost, "/payments", `{}`, "req-payment-002")
	defer response.Body.Close()

	assertPaymentErrorResponse(t, response, http.StatusBadRequest, "invalid_input", "invoice_id is required", "req-payment-002")
}

func TestCreatePaymentMapsConflictWhenPaymentAlreadyExists(t *testing.T) {
	t.Parallel()

	repository := newPaymentRepositoryStub()
	repository.byInvoice["1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe"] = "payment-001"

	server := newPaymentTestServer(t, repository, &invoicePaymentPortStub{}, paymentGatewayStub{})
	defer server.Close()

	response := performPaymentRequest(t, server, http.MethodPost, "/payments", `{"invoice_id":"1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe"}`, "req-payment-003")
	defer response.Body.Close()

	assertPaymentErrorResponse(t, response, http.StatusConflict, "payment_conflict", "payment already exists for invoice", "req-payment-003")
}

func TestCreatePaymentMapsInvoiceNotPayable(t *testing.T) {
	t.Parallel()

	server := newPaymentTestServer(t, newPaymentRepositoryStub(), &invoicePaymentPortStub{getErr: invoiceentities.ErrInvoiceNotPayable}, paymentGatewayStub{})
	defer server.Close()

	response := performPaymentRequest(t, server, http.MethodPost, "/payments", `{"invoice_id":"1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe"}`, "req-payment-004")
	defer response.Body.Close()

	assertPaymentErrorResponse(t, response, http.StatusConflict, "invoice_not_payable", "invoice is not payable", "req-payment-004")
}

func newPaymentTestServer(t *testing.T, repository *paymentRepositoryStub, invoicePort *invoicePaymentPortStub, gateway paymentGatewayStub) *httptest.Server {
	t.Helper()

	logger, err := logging.NewWithWriter("info", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	handler := NewHandler(
		usecases.NewProcessPayment(
			repository,
			invoicePort,
			gateway,
			txManagerStub{},
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
