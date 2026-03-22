package customershttp

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/rgomids/atlas-erp-core/internal/customers/application/usecases"
	"github.com/rgomids/atlas-erp-core/internal/customers/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/customers/domain/valueobjects"
	httpapi "github.com/rgomids/atlas-erp-core/internal/shared/http"
)

type Handler struct {
	createCustomer     usecases.CreateCustomer
	updateCustomer     usecases.UpdateCustomer
	deactivateCustomer usecases.DeactivateCustomer
}

func NewHandler(
	createCustomer usecases.CreateCustomer,
	updateCustomer usecases.UpdateCustomer,
	deactivateCustomer usecases.DeactivateCustomer,
) Handler {
	return Handler{
		createCustomer:     createCustomer,
		updateCustomer:     updateCustomer,
		deactivateCustomer: deactivateCustomer,
	}
}

func (handler Handler) Routes(router chi.Router) {
	router.Route("/customers", func(router chi.Router) {
		router.Post("/", handler.create)
		router.Put("/{id}", handler.update)
		router.Patch("/{id}/inactive", handler.deactivate)
	})
}

type createCustomerRequest struct {
	Name     string `json:"name"`
	Document string `json:"document"`
	Email    string `json:"email"`
}

type updateCustomerRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (handler Handler) create(writer http.ResponseWriter, request *http.Request) {
	var payload createCustomerRequest
	if err := httpapi.DecodeJSON(request, &payload); err != nil {
		httpapi.WriteInputError(writer, request, "invalid JSON payload")
		return
	}

	if err := validateCreateCustomerRequest(payload); err != nil {
		httpapi.WriteInputError(writer, request, err.Error())
		return
	}

	customer, err := handler.createCustomer.Execute(request.Context(), usecases.CreateCustomerInput{
		Name:     payload.Name,
		Document: payload.Document,
		Email:    payload.Email,
	})
	if err != nil {
		handler.writeError(writer, request, err)
		return
	}

	httpapi.WriteJSON(writer, http.StatusCreated, customer)
}

func (handler Handler) update(writer http.ResponseWriter, request *http.Request) {
	var payload updateCustomerRequest
	if err := httpapi.DecodeJSON(request, &payload); err != nil {
		httpapi.WriteInputError(writer, request, "invalid JSON payload")
		return
	}

	customerID := chi.URLParam(request, "id")
	if err := validateCustomerID(customerID); err != nil {
		httpapi.WriteInputError(writer, request, err.Error())
		return
	}

	if err := validateUpdateCustomerRequest(payload); err != nil {
		httpapi.WriteInputError(writer, request, err.Error())
		return
	}

	customer, err := handler.updateCustomer.Execute(request.Context(), usecases.UpdateCustomerInput{
		ID:    customerID,
		Name:  payload.Name,
		Email: payload.Email,
	})
	if err != nil {
		handler.writeError(writer, request, err)
		return
	}

	httpapi.WriteJSON(writer, http.StatusOK, customer)
}

func (handler Handler) deactivate(writer http.ResponseWriter, request *http.Request) {
	customerID := chi.URLParam(request, "id")
	if err := validateCustomerID(customerID); err != nil {
		httpapi.WriteInputError(writer, request, err.Error())
		return
	}

	customer, err := handler.deactivateCustomer.Execute(request.Context(), usecases.DeactivateCustomerInput{
		ID: customerID,
	})
	if err != nil {
		handler.writeError(writer, request, err)
		return
	}

	httpapi.WriteJSON(writer, http.StatusOK, customer)
}

func (handler Handler) writeError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, entities.ErrInvalidCustomerID),
		errors.Is(err, entities.ErrCustomerNameRequired),
		errors.Is(err, valueobjects.ErrInvalidDocument),
		errors.Is(err, valueobjects.ErrInvalidEmail):
		httpapi.WriteInputError(writer, request, err.Error())
	case errors.Is(err, entities.ErrCustomerAlreadyExists):
		httpapi.WriteError(writer, request, http.StatusConflict, "customer_conflict", err.Error())
	case errors.Is(err, entities.ErrCustomerNotFound):
		httpapi.WriteError(writer, request, http.StatusNotFound, "customer_not_found", err.Error())
	case errors.Is(err, entities.ErrCustomerInactive):
		httpapi.WriteError(writer, request, http.StatusConflict, "customer_inactive", err.Error())
	default:
		httpapi.WriteInternalError(writer, request, err)
	}
}

func validateCreateCustomerRequest(payload createCustomerRequest) error {
	if err := httpapi.RequireNonBlank("name", payload.Name); err != nil {
		return err
	}
	if err := httpapi.RequireNonBlank("document", payload.Document); err != nil {
		return err
	}
	if err := httpapi.RequireNonBlank("email", payload.Email); err != nil {
		return err
	}

	return nil
}

func validateUpdateCustomerRequest(payload updateCustomerRequest) error {
	if err := httpapi.RequireNonBlank("name", payload.Name); err != nil {
		return err
	}
	if err := httpapi.RequireNonBlank("email", payload.Email); err != nil {
		return err
	}

	return nil
}

func validateCustomerID(customerID string) error {
	return httpapi.RequireUUID("customer_id", customerID)
}
