package correlation

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareKeepsIncomingCorrelationID(t *testing.T) {
	t.Parallel()

	var received string

	handler := Middleware("X-Correlation-ID")(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		received = ID(request.Context())
		writer.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("X-Correlation-ID", "fixed-id")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if received != "fixed-id" {
		t.Fatalf("expected correlation ID in context, got %q", received)
	}

	if response.Header().Get("X-Correlation-ID") != "fixed-id" {
		t.Fatalf("expected response header to echo correlation id, got %q", response.Header().Get("X-Correlation-ID"))
	}
}

func TestMiddlewareGeneratesCorrelationID(t *testing.T) {
	t.Parallel()

	var received string

	handler := Middleware("X-Correlation-ID")(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		received = ID(request.Context())
		writer.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if received == "" {
		t.Fatal("expected correlation ID to be generated")
	}

	if response.Header().Get("X-Correlation-ID") == "" {
		t.Fatal("expected response header to include a generated correlation ID")
	}
}
