package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/rgomids/atlas-erp-core/internal/shared/correlation"
)

type loggerContextKey string

const requestLoggerKey loggerContextKey = "request_logger"

func bindLogger(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if base == nil {
				base = slog.Default()
			}

			logger := base.With(
				slog.String("module", moduleFromPath(request.URL.Path)),
				slog.String("request_id", correlation.ID(request.Context())),
			)

			ctx := context.WithValue(request.Context(), requestLoggerKey, logger)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

func LoggerFromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(requestLoggerKey).(*slog.Logger)
	if ok && logger != nil {
		return logger
	}

	return slog.Default().With(
		slog.String("module", "shared"),
		slog.String("request_id", correlation.ID(ctx)),
	)
}

func moduleFromPath(path string) string {
	switch {
	case strings.HasPrefix(path, "/payments"):
		return "payments"
	case strings.HasPrefix(path, "/invoices"), strings.HasSuffix(path, "/invoices"):
		return "invoices"
	case strings.HasPrefix(path, "/customers"):
		return "customers"
	case strings.HasPrefix(path, "/health"):
		return "shared"
	default:
		return "shared"
	}
}
