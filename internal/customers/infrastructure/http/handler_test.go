package customershttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rgomids/atlas-erp-core/internal/customers/application/usecases"
	"github.com/rgomids/atlas-erp-core/internal/customers/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/customers/domain/repositories"
	httpapi "github.com/rgomids/atlas-erp-core/internal/shared/http"
	"github.com/rgomids/atlas-erp-core/internal/shared/logging"
)

type customerRepositoryStub struct {
	byID             map[string]entities.Customer
	byDocument       map[string]string
	existsByDocument bool
}

var _ repositories.CustomerRepository = (*customerRepositoryStub)(nil)

func newCustomerRepositoryStub() *customerRepositoryStub {
	return &customerRepositoryStub{
		byID:       map[string]entities.Customer{},
		byDocument: map[string]string{},
	}
}

func (repository *customerRepositoryStub) ExistsByDocument(_ context.Context, _ string) (bool, error) {
	return repository.existsByDocument, nil
}

func (repository *customerRepositoryStub) Save(_ context.Context, customer entities.Customer) error {
	repository.byID[customer.ID()] = customer
	repository.byDocument[customer.Document().Value()] = customer.ID()
	return nil
}

func (repository *customerRepositoryStub) GetByID(_ context.Context, customerID string) (entities.Customer, error) {
	customer, ok := repository.byID[customerID]
	if !ok {
		return entities.Customer{}, entities.ErrCustomerNotFound
	}

	return customer, nil
}

func (repository *customerRepositoryStub) Update(_ context.Context, customer entities.Customer) error {
	if _, ok := repository.byID[customer.ID()]; !ok {
		return entities.ErrCustomerNotFound
	}

	repository.byID[customer.ID()] = customer
	return nil
}

func TestCreateCustomerReturnsCreatedPayload(t *testing.T) {
	t.Parallel()

	server := newCustomerTestServer(t, newCustomerRepositoryStub())
	defer server.Close()

	response := performCustomerRequest(t, server, http.MethodPost, "/customers", `{"name":"Atlas Co","document":"12345678900","email":"team@atlas.io"}`, "req-customer-001")
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.StatusCode)
	}

	var payload struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode customer payload: %v", err)
	}

	if payload.ID == "" {
		t.Fatal("expected customer id to be generated")
	}

	if payload.Status != "Active" {
		t.Fatalf("expected active customer, got %q", payload.Status)
	}
}

func TestCreateCustomerRejectsMissingDocumentAtHTTPBoundary(t *testing.T) {
	t.Parallel()

	server := newCustomerTestServer(t, newCustomerRepositoryStub())
	defer server.Close()

	response := performCustomerRequest(t, server, http.MethodPost, "/customers", `{"name":"Atlas Co","email":"team@atlas.io"}`, "req-customer-002")
	defer response.Body.Close()

	assertCustomerErrorResponse(t, response, http.StatusBadRequest, "invalid_input", "document is required", "req-customer-002")
}

func TestCreateCustomerMapsDuplicateDocumentToConflict(t *testing.T) {
	t.Parallel()

	repository := newCustomerRepositoryStub()
	repository.existsByDocument = true

	server := newCustomerTestServer(t, repository)
	defer server.Close()

	response := performCustomerRequest(t, server, http.MethodPost, "/customers", `{"name":"Atlas Co","document":"12345678900","email":"team@atlas.io"}`, "req-customer-003")
	defer response.Body.Close()

	assertCustomerErrorResponse(t, response, http.StatusConflict, "customer_conflict", "customer already exists", "req-customer-003")
}

func TestUpdateCustomerRejectsInvalidCustomerID(t *testing.T) {
	t.Parallel()

	server := newCustomerTestServer(t, newCustomerRepositoryStub())
	defer server.Close()

	response := performCustomerRequest(t, server, http.MethodPut, "/customers/not-a-uuid", `{"name":"Atlas Updated","email":"billing@atlas.io"}`, "req-customer-004")
	defer response.Body.Close()

	assertCustomerErrorResponse(t, response, http.StatusBadRequest, "invalid_input", "customer_id must be a valid UUID", "req-customer-004")
}

func newCustomerTestServer(t *testing.T, repository *customerRepositoryStub) *httptest.Server {
	t.Helper()

	logger, err := logging.NewWithWriter("info", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	handler := NewHandler(
		usecases.NewCreateCustomer(repository),
		usecases.NewUpdateCustomer(repository),
		usecases.NewDeactivateCustomer(repository),
	)

	return httptest.NewServer(httpapi.NewRouter(logger, "X-Correlation-ID", handler.Routes))
}

func performCustomerRequest(t *testing.T, server *httptest.Server, method, path, payload, requestID string) *http.Response {
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

func assertCustomerErrorResponse(t *testing.T, response *http.Response, expectedStatus int, expectedError string, expectedMessage string, expectedRequestID string) {
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
