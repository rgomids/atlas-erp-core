package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/rgomids/atlas-erp-core/internal/shared/correlation"
)

type ErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func WriteError(writer http.ResponseWriter, request *http.Request, statusCode int, errorCode, message string) {
	WriteJSON(writer, statusCode, ErrorResponse{
		Error:     errorCode,
		Message:   message,
		RequestID: correlation.ID(request.Context()),
	})
}

func WriteInputError(writer http.ResponseWriter, request *http.Request, message string) {
	WriteError(writer, request, http.StatusBadRequest, "invalid_input", message)
}

func WriteInternalError(writer http.ResponseWriter, request *http.Request, err error) {
	LoggerFromContext(request.Context()).Error(
		"unexpected request failure",
		slog.Any("err", err),
	)

	WriteError(writer, request, http.StatusInternalServerError, "internal_error", "internal server error")
}
