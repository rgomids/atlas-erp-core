package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/rgomids/atlas-erp-core/internal/shared/correlation"
)

func NewRouter(logger *slog.Logger, correlationHeader string) http.Handler {
	router := chi.NewRouter()
	router.Use(correlation.Middleware(correlationHeader))
	router.Use(requestLogger(logger))
	router.Get("/health", healthHandler)

	return router
}

func healthHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(`{"status":"ok"}`))
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			recorder := newStatusRecorder(writer)
			startedAt := time.Now()

			next.ServeHTTP(recorder, request)

			logger.Info(
				"http request completed",
				slog.String("method", request.Method),
				slog.String("path", request.URL.Path),
				slog.Int("status_code", recorder.statusCode),
				slog.Duration("duration", time.Since(startedAt)),
				slog.String("correlation_id", correlation.ID(request.Context())),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
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
