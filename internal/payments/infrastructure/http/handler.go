package paymentshttp

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	invoiceentities "github.com/rgomids/atlas-erp-core/internal/invoices/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/payments/application/usecases"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
	httpapi "github.com/rgomids/atlas-erp-core/internal/shared/http"
)

type Handler struct {
	processPayment usecases.ProcessPayment
}

func NewHandler(processPayment usecases.ProcessPayment) Handler {
	return Handler{processPayment: processPayment}
}

func (handler Handler) Routes(router chi.Router) {
	router.Post("/payments", handler.create)
}

type processPaymentRequest struct {
	InvoiceID string `json:"invoice_id"`
}

func (handler Handler) create(writer http.ResponseWriter, request *http.Request) {
	var payload processPaymentRequest
	if err := httpapi.DecodeJSON(request, &payload); err != nil {
		httpapi.WriteError(writer, request, http.StatusBadRequest, "invalid_request", "invalid JSON payload")
		return
	}

	payment, err := handler.processPayment.Execute(request.Context(), usecases.ProcessPaymentInput{
		InvoiceID: payload.InvoiceID,
	})
	if err != nil {
		handler.writeError(writer, request, err)
		return
	}

	httpapi.WriteJSON(writer, http.StatusCreated, payment)
}

func (handler Handler) writeError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, entities.ErrInvalidInvoiceReference):
		httpapi.WriteError(writer, request, http.StatusBadRequest, "invalid_payment", err.Error())
	case errors.Is(err, entities.ErrPaymentAlreadyExists):
		httpapi.WriteError(writer, request, http.StatusConflict, "payment_conflict", err.Error())
	case errors.Is(err, invoiceentities.ErrInvoiceNotFound):
		httpapi.WriteError(writer, request, http.StatusNotFound, "invoice_not_found", err.Error())
	case errors.Is(err, invoiceentities.ErrInvoiceImmutable),
		errors.Is(err, invoiceentities.ErrInvoiceNotPayable):
		httpapi.WriteError(writer, request, http.StatusConflict, "invoice_not_payable", err.Error())
	default:
		httpapi.WriteError(writer, request, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
