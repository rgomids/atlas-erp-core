package functional_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	httpapi "github.com/rgomids/atlas-erp-core/internal/shared/http"
	"github.com/rgomids/atlas-erp-core/internal/shared/logging"
	"github.com/rgomids/atlas-erp-core/internal/shared/postgres"
	"github.com/rgomids/atlas-erp-core/test/support"
)

func TestHealthEndpointRespondsWithFoundationContract(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	databaseConfig, cleanup := support.StartPostgres(ctx, t)
	defer cleanup()

	pool, err := postgres.Open(ctx, databaseConfig)
	if err != nil {
		t.Fatalf("open postgres pool: %v", err)
	}
	defer pool.Close()

	logger, err := logging.NewWithWriter("info", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	server := httptest.NewServer(httpapi.NewRouter(logger, "X-Correlation-ID"))
	defer server.Close()

	response, err := server.Client().Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("perform health request: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	if string(body) != `{"status":"ok"}` {
		t.Fatalf("expected exact health payload, got %q", string(body))
	}
}
