package invoiceshttp

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	customerentities "github.com/rgomids/atlas-erp-core/internal/customers/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/invoices/application/usecases"
	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/entities"
	httpapi "github.com/rgomids/atlas-erp-core/internal/shared/http"
)

type Handler struct {
	createInvoice       usecases.CreateInvoice
	listCustomerInvoice usecases.ListCustomerInvoices
}

func NewHandler(
	createInvoice usecases.CreateInvoice,
	listCustomerInvoice usecases.ListCustomerInvoices,
) Handler {
	return Handler{
		createInvoice:       createInvoice,
		listCustomerInvoice: listCustomerInvoice,
	}
}

func (handler Handler) Routes(router chi.Router) {
	router.Post("/invoices", handler.create)
	router.Get("/customers/{id}/invoices", handler.listByCustomer)
}

type createInvoiceRequest struct {
	CustomerID  string `json:"customer_id"`
	AmountCents int64  `json:"amount_cents"`
	DueDate     string `json:"due_date"`
}

func (handler Handler) create(writer http.ResponseWriter, request *http.Request) {
	var payload createInvoiceRequest
	if err := httpapi.DecodeJSON(request, &payload); err != nil {
		httpapi.WriteError(writer, request, http.StatusBadRequest, "invalid_request", "invalid JSON payload")
		return
	}

	invoice, err := handler.createInvoice.Execute(request.Context(), usecases.CreateInvoiceInput{
		CustomerID:  payload.CustomerID,
		AmountCents: payload.AmountCents,
		DueDate:     payload.DueDate,
	})
	if err != nil {
		handler.writeError(writer, request, err)
		return
	}

	httpapi.WriteJSON(writer, http.StatusCreated, invoice)
}

func (handler Handler) listByCustomer(writer http.ResponseWriter, request *http.Request) {
	invoices, err := handler.listCustomerInvoice.Execute(request.Context(), usecases.ListCustomerInvoicesInput{
		CustomerID: chi.URLParam(request, "id"),
	})
	if err != nil {
		handler.writeError(writer, request, err)
		return
	}

	httpapi.WriteJSON(writer, http.StatusOK, map[string]any{"items": invoices})
}

func (handler Handler) writeError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, entities.ErrInvalidCustomerReference),
		errors.Is(err, entities.ErrInvoiceAmountMustBePositive),
		errors.Is(err, entities.ErrInvoiceDueDateRequired):
		httpapi.WriteError(writer, request, http.StatusBadRequest, "invalid_invoice", err.Error())
	case errors.Is(err, customerentities.ErrCustomerNotFound):
		httpapi.WriteError(writer, request, http.StatusNotFound, "customer_not_found", err.Error())
	case errors.Is(err, customerentities.ErrCustomerInactive):
		httpapi.WriteError(writer, request, http.StatusConflict, "customer_inactive", err.Error())
	case errors.Is(err, entities.ErrInvoiceNotFound):
		httpapi.WriteError(writer, request, http.StatusNotFound, "invoice_not_found", err.Error())
	default:
		httpapi.WriteError(writer, request, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
