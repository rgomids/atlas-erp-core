package correlation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

type contextKey string

const correlationIDKey contextKey = "correlation_id"

func Middleware(header string) func(http.Handler) http.Handler {
	headerName := strings.TrimSpace(header)
	if headerName == "" {
		headerName = "X-Correlation-ID"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			correlationID := strings.TrimSpace(request.Header.Get(headerName))
			if correlationID == "" {
				correlationID = generateID()
			}

			writer.Header().Set(headerName, correlationID)

			ctx := context.WithValue(request.Context(), correlationIDKey, correlationID)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

func ID(ctx context.Context) string {
	value, ok := ctx.Value(correlationIDKey).(string)
	if !ok {
		return ""
	}

	return value
}

func generateID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "correlation-id-unavailable"
	}

	return hex.EncodeToString(buffer)
}
