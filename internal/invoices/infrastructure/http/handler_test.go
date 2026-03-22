package invoiceshttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	customerports "github.com/rgomids/atlas-erp-core/internal/customers/application/ports"
	customerentities "github.com/rgomids/atlas-erp-core/internal/customers/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/invoices/application/usecases"
	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/repositories"
	httpapi "github.com/rgomids/atlas-erp-core/internal/shared/http"
	"github.com/rgomids/atlas-erp-core/internal/shared/logging"
)

type invoiceRepositoryStub struct {
	byID       map[string]entities.Invoice
	byCustomer map[string][]entities.Invoice
}

var _ repositories.InvoiceRepository = (*invoiceRepositoryStub)(nil)

func newInvoiceRepositoryStub() *invoiceRepositoryStub {
	return &invoiceRepositoryStub{
		byID:       map[string]entities.Invoice{},
		byCustomer: map[string][]entities.Invoice{},
	}
}

func (repository *invoiceRepositoryStub) Save(_ context.Context, invoice entities.Invoice) error {
	repository.byID[invoice.ID()] = invoice
	repository.byCustomer[invoice.CustomerID()] = append(repository.byCustomer[invoice.CustomerID()], invoice)
	return nil
}

func (repository *invoiceRepositoryStub) GetByID(_ context.Context, invoiceID string) (entities.Invoice, error) {
	invoice, ok := repository.byID[invoiceID]
	if !ok {
		return entities.Invoice{}, entities.ErrInvoiceNotFound
	}

	return invoice, nil
}

func (repository *invoiceRepositoryStub) ListByCustomerID(_ context.Context, customerID string) ([]entities.Invoice, error) {
	return repository.byCustomer[customerID], nil
}

func (repository *invoiceRepositoryStub) Update(_ context.Context, invoice entities.Invoice) error {
	repository.byID[invoice.ID()] = invoice
	return nil
}

type existenceCheckerStub struct {
	err error
}

var _ customerports.ExistenceChecker = (*existenceCheckerStub)(nil)

func (checker existenceCheckerStub) ExistsActiveCustomer(context.Context, string) error {
	return checker.err
}

func TestCreateInvoiceReturnsCreatedPayload(t *testing.T) {
	t.Parallel()

	server := newInvoiceTestServer(t, newInvoiceRepositoryStub(), existenceCheckerStub{})
	defer server.Close()

	response := performInvoiceRequest(t, server, http.MethodPost, "/invoices", `{"customer_id":"1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe","amount_cents":1599,"due_date":"2026-03-25"}`, "req-invoice-001")
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.StatusCode)
	}

	var payload struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode invoice payload: %v", err)
	}

	if payload.ID == "" {
		t.Fatal("expected invoice id to be generated")
	}

	if payload.Status != "Pending" {
		t.Fatalf("expected pending invoice, got %q", payload.Status)
	}
}

func TestCreateInvoiceRejectsInvalidDateAtHTTPBoundary(t *testing.T) {
	t.Parallel()

	server := newInvoiceTestServer(t, newInvoiceRepositoryStub(), existenceCheckerStub{})
	defer server.Close()

	response := performInvoiceRequest(t, server, http.MethodPost, "/invoices", `{"customer_id":"1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe","amount_cents":1599,"due_date":"25/03/2026"}`, "req-invoice-002")
	defer response.Body.Close()

	assertInvoiceErrorResponse(t, response, http.StatusBadRequest, "invalid_input", "due_date must use YYYY-MM-DD", "req-invoice-002")
}

func TestCreateInvoiceMapsCustomerNotFound(t *testing.T) {
	t.Parallel()

	server := newInvoiceTestServer(t, newInvoiceRepositoryStub(), existenceCheckerStub{err: customerentities.ErrCustomerNotFound})
	defer server.Close()

	response := performInvoiceRequest(t, server, http.MethodPost, "/invoices", `{"customer_id":"1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe","amount_cents":1599,"due_date":"2026-03-25"}`, "req-invoice-003")
	defer response.Body.Close()

	assertInvoiceErrorResponse(t, response, http.StatusNotFound, "customer_not_found", "customer not found", "req-invoice-003")
}

func TestListInvoicesRejectsInvalidCustomerID(t *testing.T) {
	t.Parallel()

	server := newInvoiceTestServer(t, newInvoiceRepositoryStub(), existenceCheckerStub{})
	defer server.Close()

	response := performInvoiceRequest(t, server, http.MethodGet, "/customers/not-a-uuid/invoices", "", "req-invoice-004")
	defer response.Body.Close()

	assertInvoiceErrorResponse(t, response, http.StatusBadRequest, "invalid_input", "customer_id must be a valid UUID", "req-invoice-004")
}

func newInvoiceTestServer(t *testing.T, repository *invoiceRepositoryStub, checker existenceCheckerStub) *httptest.Server {
	t.Helper()

	logger, err := logging.NewWithWriter("info", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	handler := NewHandler(
		usecases.NewCreateInvoice(repository, checker),
		usecases.NewListCustomerInvoices(repository),
	)

	return httptest.NewServer(httpapi.NewRouter(logger, "X-Correlation-ID", handler.Routes))
}

func performInvoiceRequest(t *testing.T, server *httptest.Server, method, path, payload, requestID string) *http.Response {
	t.Helper()

	var body *strings.Reader
	if payload == "" {
		body = strings.NewReader("")
	} else {
		body = strings.NewReader(payload)
	}

	request, err := http.NewRequest(method, server.URL+path, body)
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

func assertInvoiceErrorResponse(t *testing.T, response *http.Response, expectedStatus int, expectedError string, expectedMessage string, expectedRequestID string) {
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
