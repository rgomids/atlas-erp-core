package observability

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
)

func RoutePattern(request *http.Request) string {
	if request == nil {
		return "/"
	}

	if routeContext := chi.RouteContext(request.Context()); routeContext != nil {
		pattern := strings.TrimSpace(routeContext.RoutePattern())
		if pattern != "" {
			return pattern
		}
	}

	if request.URL == nil || strings.TrimSpace(request.URL.Path) == "" {
		return "/"
	}

	return request.URL.Path
}

func HTTPSpanName(method string, route string) string {
	return fmt.Sprintf("http.request %s %s", strings.ToUpper(strings.TrimSpace(method)), route)
}

func HTTPSpanAttributes(request *http.Request, route string, statusCode int, module string, errorType string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("http.method", normalizeMetricValue(request.Method, "GET")),
		attribute.String("http.route", normalizeMetricValue(route, "/")),
		attribute.Int("http.response.status_code", statusCode),
		attribute.String("atlas.module", normalizeMetricValue(module, "shared")),
	}

	if errorType != "" {
		attrs = append(attrs, attribute.String("error.type", errorType))
	}

	return attrs
}
