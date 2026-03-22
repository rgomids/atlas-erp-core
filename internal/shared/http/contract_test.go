package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/rgomids/atlas-erp-core/internal/shared/logging"
)

func TestWriteInputErrorUsesCanonicalContract(t *testing.T) {
	t.Parallel()

	logger, err := logging.NewWithWriter("info", &bytes.Buffer{})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	router := NewRouter(logger, "X-Correlation-ID", func(router chi.Router) {
		router.Get("/customers/error", func(writer http.ResponseWriter, request *http.Request) {
			WriteInputError(writer, request, "document is required")
		})
	})

	request := httptest.NewRequest(http.MethodGet, "/customers/error", nil)
	request.Header.Set("X-Correlation-ID", "req-123")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}

	if response.Header().Get("X-Correlation-ID") != "req-123" {
		t.Fatalf("expected correlation header to be echoed, got %q", response.Header().Get("X-Correlation-ID"))
	}

	var payload ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}

	if payload.Error != "invalid_input" {
		t.Fatalf("expected invalid_input error code, got %q", payload.Error)
	}

	if payload.Message != "document is required" {
		t.Fatalf("expected validation message, got %q", payload.Message)
	}

	if payload.RequestID != "req-123" {
		t.Fatalf("expected request_id to match correlation id, got %q", payload.RequestID)
	}
}

func TestRouterLogsModuleAndRequestID(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	logger, err := logging.NewWithWriter("info", buffer)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	router := NewRouter(logger, "X-Correlation-ID", func(router chi.Router) {
		router.Get("/customers/123", func(writer http.ResponseWriter, _ *http.Request) {
			WriteJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
		})
	})

	request := httptest.NewRequest(http.MethodGet, "/customers/123", nil)
	request.Header.Set("X-Correlation-ID", "req-456")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	output := buffer.String()
	for _, fragment := range []string{
		`"msg":"http request completed"`,
		`"module":"customers"`,
		`"request_id":"req-456"`,
		`"status_code":200`,
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected log output to contain %s, got %s", fragment, output)
		}
	}
}

func TestModuleFromPathInfersExpectedDomain(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		"/health":                                "shared",
		"/customers":                             "customers",
		"/customers/1adf3d42-7b1d-4d2b-a7d6/foo": "customers",
		"/invoices":                              "invoices",
		"/customers/1adf3d42-7b1d-4d2b-a7d6/invoices": "invoices",
		"/payments": "payments",
		"/unknown":  "shared",
	}

	for path, expectedModule := range testCases {
		path := path
		expectedModule := expectedModule

		t.Run(path, func(t *testing.T) {
			t.Parallel()

			if got := moduleFromPath(path); got != expectedModule {
				t.Fatalf("expected module %q, got %q", expectedModule, got)
			}
		})
	}
}
