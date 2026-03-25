package benchmark

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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

var (
	reportJSONPath     = flag.String("report-json", "", "write benchmark report to JSON")
	reportMarkdownPath = flag.String("report-md", "", "write benchmark report to Markdown")
	benchmarkReports   = newRegistry()
)

func TestMain(m *testing.M) {
	flag.Parse()

	code := m.Run()

	if code == 0 {
		if err := writeReports(benchmarkReports, *reportJSONPath, *reportMarkdownPath, time.Now()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			code = 1
		}
	}

	os.Exit(code)
}

func BenchmarkCreateCustomer(b *testing.B) {
	env := newBenchmarkEnvironment(b)
	server := newBenchmarkServer(b, env.pool, config.FaultProfileNone, integration.NewMockGateway(), env.timeout)
	defer server.Close()

	collector := newCollector("BenchmarkCreateCustomer")
	client := server.Client()

	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		startedAt := time.Now()
		_, err := createCustomer(client, server.URL, iteration)
		collector.Record(time.Since(startedAt), err)
	}
	b.StopTimer()

	reportBenchmarkMetrics(b, collector)
}

func BenchmarkCreateInvoice(b *testing.B) {
	env := newBenchmarkEnvironment(b)
	server := newBenchmarkServer(b, env.pool, config.FaultProfileNone, integration.NewMockGateway(), env.timeout)
	defer server.Close()

	client := server.Client()
	customerID, err := createCustomer(client, server.URL, 0)
	if err != nil {
		b.Fatalf("seed customer for invoice benchmark: %v", err)
	}

	collector := newCollector("BenchmarkCreateInvoice")

	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		startedAt := time.Now()
		_, err := createInvoice(client, server.URL, customerID, iteration)
		collector.Record(time.Since(startedAt), err)
	}
	b.StopTimer()

	reportBenchmarkMetrics(b, collector)
}

func BenchmarkProcessPaymentRetry(b *testing.B) {
	env := newBenchmarkEnvironment(b)
	collector := newCollector("BenchmarkProcessPaymentRetry")

	for iteration := 0; iteration < b.N; iteration++ {
		failedServer := newBenchmarkServer(b, env.pool, config.FaultProfileNone, integration.NewMockGatewayWithStatus("Failed"), env.timeout)
		approvedServer := newBenchmarkServer(b, env.pool, config.FaultProfileNone, integration.NewMockGateway(), env.timeout)

		customerID, err := createCustomer(failedServer.Client(), failedServer.URL, iteration)
		if err != nil {
			failedServer.Close()
			approvedServer.Close()
			b.Fatalf("seed customer for retry benchmark: %v", err)
		}

		invoiceID, err := createInvoice(failedServer.Client(), failedServer.URL, customerID, iteration)
		if err != nil {
			failedServer.Close()
			approvedServer.Close()
			b.Fatalf("seed failed invoice for retry benchmark: %v", err)
		}

		b.StartTimer()
		startedAt := time.Now()
		err = retryPayment(approvedServer.Client(), approvedServer.URL, invoiceID)
		collector.Record(time.Since(startedAt), err)
		b.StopTimer()

		failedServer.Close()
		approvedServer.Close()
	}

	reportBenchmarkMetrics(b, collector)
}

func BenchmarkEndToEndFlow(b *testing.B) {
	env := newBenchmarkEnvironment(b)
	server := newBenchmarkServer(b, env.pool, config.FaultProfileNone, integration.NewMockGateway(), env.timeout)
	defer server.Close()

	client := server.Client()
	collector := newCollector("BenchmarkEndToEndFlow")

	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		startedAt := time.Now()
		customerID, err := createCustomer(client, server.URL, iteration)
		if err == nil {
			var invoiceID string
			invoiceID, err = createInvoice(client, server.URL, customerID, iteration)
			if err == nil {
				err = assertCustomerInvoicePaid(client, server.URL, customerID, invoiceID)
			}
		}

		collector.Record(time.Since(startedAt), err)
	}
	b.StopTimer()

	reportBenchmarkMetrics(b, collector)
}

type benchmarkEnvironment struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func newBenchmarkEnvironment(b *testing.B) benchmarkEnvironment {
	b.Helper()

	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, b)
	support.RunMigrations(b, databaseConfig)

	pool, err := sharedpostgres.Open(ctx, databaseConfig)
	if err != nil {
		cleanup()
		b.Fatalf("open postgres for benchmark: %v", err)
	}

	b.Cleanup(func() {
		pool.Close()
		cleanup()
	})

	return benchmarkEnvironment{
		pool:    pool,
		timeout: time.Second,
	}
}

func newBenchmarkServer(
	b *testing.B,
	pool *pgxpool.Pool,
	profile config.FaultProfile,
	gateway paymentports.PaymentGateway,
	timeout time.Duration,
) *httptest.Server {
	b.Helper()

	logger, err := logging.NewWithWriter("error", io.Discard)
	if err != nil {
		b.Fatalf("create benchmark logger: %v", err)
	}

	recorder := runtimefaults.DecorateRecorder(profile, outbox.NewPostgresRecorder(pool))
	eventBus := sharedevent.NewSyncBusWithOptions(runtimefaults.EventBusOptions(profile, nil, recorder))
	customerModule := customers.NewModule(pool, eventBus)
	invoiceModule := invoices.NewModule(pool, customerModule.ExistenceChecker(), eventBus)
	billingModule := billing.NewModule(pool, eventBus)
	paymentModule := payments.NewModule(pool, billingModule.PaymentPort(), eventBus, runtimefaults.DecorateGateway(profile, timeout, gateway), payments.ModuleConfig{
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

func reportBenchmarkMetrics(b *testing.B, collector *collector) {
	summary := collector.Summary()

	b.ReportMetric(summary.AvgMS, "avg_ms")
	b.ReportMetric(summary.P95MS, "p95_ms")
	b.ReportMetric(summary.OpsPerSec, "ops/s")
	b.ReportMetric(summary.ErrorRatePct, "error_rate_pct")

	benchmarkReports.Record(summary)
}

func createCustomer(client *http.Client, baseURL string, iteration int) (string, error) {
	response, err := postJSON(
		client,
		baseURL+"/customers",
		fmt.Sprintf(`{"name":"Atlas Co %d","document":"%s","email":"bench-%d@atlas.io"}`, iteration, benchmarkDocument(iteration), iteration),
	)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create customer returned status %d", response.StatusCode)
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode create customer response: %w", err)
	}

	return payload.ID, nil
}

func createInvoice(client *http.Client, baseURL string, customerID string, iteration int) (string, error) {
	response, err := postJSON(
		client,
		baseURL+"/invoices",
		fmt.Sprintf(`{"customer_id":"%s","amount_cents":%d,"due_date":"2026-03-31"}`, customerID, 1500+iteration),
	)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create invoice returned status %d", response.StatusCode)
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode create invoice response: %w", err)
	}

	return payload.ID, nil
}

func retryPayment(client *http.Client, baseURL string, invoiceID string) error {
	response, err := postJSON(
		client,
		baseURL+"/payments",
		fmt.Sprintf(`{"invoice_id":"%s"}`, invoiceID),
	)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return fmt.Errorf("retry payment returned status %d", response.StatusCode)
	}

	var payload struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return fmt.Errorf("decode retry payment response: %w", err)
	}

	if payload.Status != "Approved" {
		return fmt.Errorf("retry payment returned status %q", payload.Status)
	}

	return nil
}

func assertCustomerInvoicePaid(client *http.Client, baseURL string, customerID string, invoiceID string) error {
	response, err := client.Get(baseURL + "/customers/" + customerID + "/invoices")
	if err != nil {
		return fmt.Errorf("list invoices: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("list invoices returned status %d", response.StatusCode)
	}

	var payload struct {
		Items []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return fmt.Errorf("decode invoice list response: %w", err)
	}

	for _, item := range payload.Items {
		if item.ID == invoiceID && item.Status == "Paid" {
			return nil
		}
	}

	return fmt.Errorf("invoice %s was not paid", invoiceID)
}

func postJSON(client *http.Client, url string, payload string) (*http.Response, error) {
	request, err := http.NewRequest(http.MethodPost, url, strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request for %s: %w", url, err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Correlation-ID", "bench-request")

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("post %s: %w", url, err)
	}

	return response, nil
}

func benchmarkDocument(iteration int) string {
	return fmt.Sprintf("9%010d", iteration+1)
}
