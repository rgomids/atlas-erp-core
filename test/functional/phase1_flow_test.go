package functional_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rgomids/atlas-erp-core/internal/customers"
	"github.com/rgomids/atlas-erp-core/internal/invoices"
	"github.com/rgomids/atlas-erp-core/internal/payments"
	httpapi "github.com/rgomids/atlas-erp-core/internal/shared/http"
	"github.com/rgomids/atlas-erp-core/internal/shared/logging"
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
	"github.com/rgomids/atlas-erp-core/test/support"
)

func TestPhase1HTTPFlowCompletesEndToEnd(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	support.RunMigrations(t, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer pool.Close()

	logger, err := logging.NewWithWriter("info", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	customerModule := customers.NewModule(pool)
	invoiceModule := invoices.NewModule(pool, customerModule.ExistenceChecker())
	paymentModule := payments.NewModule(pool, invoiceModule.PaymentPort())

	server := httptest.NewServer(httpapi.NewRouter(
		logger,
		"X-Correlation-ID",
		customerModule.Routes,
		invoiceModule.Routes,
		paymentModule.Routes,
	))
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
		t.Fatalf("expected pending invoice, got %q", invoice.Status)
	}

	paymentResponse := postJSON(t, server.URL+"/payments", `{"invoice_id":"`+invoice.ID+`"}`)
	if paymentResponse.StatusCode != http.StatusCreated {
		t.Fatalf("expected payment creation status 201, got %d", paymentResponse.StatusCode)
	}

	var payment struct {
		Status string `json:"status"`
	}
	decodeResponse(t, paymentResponse, &payment)
	if payment.Status != "Approved" {
		t.Fatalf("expected approved payment, got %q", payment.Status)
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
		t.Fatalf("expected paid invoice after payment, got %q", invoicesPayload.Items[0].Status)
	}
}

func postJSON(t *testing.T, url string, payload string) *http.Response {
	t.Helper()

	response, err := http.Post(url, "application/json", strings.NewReader(payload))
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
