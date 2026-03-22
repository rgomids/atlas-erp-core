package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/rgomids/atlas-erp-core/internal/shared/correlation"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
)

type ErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func WriteError(writer http.ResponseWriter, request *http.Request, statusCode int, errorCode, message string) {
	writeErrorWithType(writer, request, statusCode, errorCode, message, "")
}

func WriteDomainError(writer http.ResponseWriter, request *http.Request, statusCode int, errorCode, message string) {
	writeErrorWithType(writer, request, statusCode, errorCode, message, observability.ErrorTypeDomain)
}

func writeErrorWithType(writer http.ResponseWriter, request *http.Request, statusCode int, errorCode, message string, errorType string) {
	if recorder, ok := writer.(interface{ SetErrorType(string) }); ok {
		recorder.SetErrorType(errorType)
	}

	WriteJSON(writer, statusCode, ErrorResponse{
		Error:     errorCode,
		Message:   message,
		RequestID: correlation.ID(request.Context()),
	})
}

func WriteInputError(writer http.ResponseWriter, request *http.Request, message string) {
	writeErrorWithType(writer, request, http.StatusBadRequest, "invalid_input", message, observability.ErrorTypeValidation)
}

func WriteInternalError(writer http.ResponseWriter, request *http.Request, err error) {
	LoggerFromContext(request.Context()).Error(
		"unexpected request failure",
		slog.String("error_type", observability.ErrorTypeInfrastructure),
		slog.Any("err", err),
	)

	writeErrorWithType(
		writer,
		request,
		http.StatusInternalServerError,
		"internal_error",
		"internal server error",
		observability.ErrorTypeInfrastructure,
	)
}
