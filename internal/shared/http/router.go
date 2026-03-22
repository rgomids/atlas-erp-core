package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/rgomids/atlas-erp-core/internal/shared/correlation"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
)

type RouteRegistrar func(chi.Router)

func NewRouter(logger *slog.Logger, correlationHeader string, registrars ...RouteRegistrar) http.Handler {
	return NewRouterWithObservability(logger, correlationHeader, observability.NewNoop(), registrars...)
}

func NewRouterWithObservability(logger *slog.Logger, correlationHeader string, telemetry *observability.Runtime, registrars ...RouteRegistrar) http.Handler {
	router := chi.NewRouter()
	router.Use(correlation.Middleware(correlationHeader))
	router.Use(otelhttp.NewMiddleware(
		"http.request",
		otelhttp.WithTracerProvider(telemetry.TracerProvider()),
		otelhttp.WithPropagators(telemetry.Propagator()),
	))
	router.Use(bindLogger(logger))
	router.Use(requestLogger(telemetry))
	router.Get("/health", healthHandler)
	router.Method(http.MethodGet, "/metrics", telemetry.MetricsHandler())
	for _, registrar := range registrars {
		registrar(router)
	}

	return router
}

func healthHandler(writer http.ResponseWriter, _ *http.Request) {
	WriteJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func requestLogger(telemetry *observability.Runtime) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			recorder := newStatusRecorder(writer)
			startedAt := time.Now()

			next.ServeHTTP(recorder, request)

			route := observability.RoutePattern(request)
			module := moduleFromPath(route)
			duration := time.Since(startedAt)
			span := trace.SpanFromContext(request.Context())
			span.SetName(observability.HTTPSpanName(request.Method, route))
			span.SetAttributes(observability.HTTPSpanAttributes(request, route, recorder.statusCode, module, recorder.errorType)...)
			if recorder.statusCode >= http.StatusBadRequest {
				span.SetStatus(codes.Error, recorder.errorType)
			} else {
				span.SetStatus(codes.Ok, "")
			}

			telemetry.RecordHTTPRequest(request.Context(), request.Method, route, recorder.statusCode, module, recorder.errorType, duration)

			LoggerFromContext(request.Context()).Info(
				"http request completed",
				slog.String("method", request.Method),
				slog.String("path", request.URL.Path),
				slog.String("route", route),
				slog.String("module", module),
				slog.Int("status_code", recorder.statusCode),
				slog.Duration("duration", duration),
				slog.String("error_type", recorder.errorType),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
	errorType  string
}

func newStatusRecorder(writer http.ResponseWriter) *statusRecorder {
	return &statusRecorder{
		ResponseWriter: writer,
		statusCode:     http.StatusOK,
	}
}

func (recorder *statusRecorder) WriteHeader(statusCode int) {
	recorder.statusCode = statusCode
	recorder.ResponseWriter.WriteHeader(statusCode)
}

func (recorder *statusRecorder) SetErrorType(errorType string) {
	recorder.errorType = errorType
}
