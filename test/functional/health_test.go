package functional_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	httpapi "github.com/rgomids/atlas-erp-core/internal/shared/http"
	"github.com/rgomids/atlas-erp-core/internal/shared/logging"
)

func TestHealthEndpointRespondsWithFoundationContract(t *testing.T) {
	t.Parallel()

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

	var payload map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("expected valid json response, got error: %v", err)
	}

	if payload["status"] != "ok" {
		t.Fatalf("expected status payload to be ok, got %q", payload["status"])
	}
}
