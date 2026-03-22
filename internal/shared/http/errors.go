package httpapi

import (
	"net/http"

	"github.com/rgomids/atlas-erp-core/internal/shared/correlation"
)

type ErrorResponse struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id"`
}

func WriteError(writer http.ResponseWriter, request *http.Request, statusCode int, code, message string) {
	WriteJSON(writer, statusCode, ErrorResponse{
		Code:          code,
		Message:       message,
		CorrelationID: correlation.ID(request.Context()),
	})
}
