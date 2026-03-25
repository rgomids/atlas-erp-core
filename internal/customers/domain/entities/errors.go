package entities

import (
	"errors"

	customerpublic "github.com/rgomids/atlas-erp-core/internal/customers/public"
)

var (
	ErrInvalidCustomerID     = errors.New("invalid customer id")
	ErrCustomerNameRequired  = errors.New("customer name is required")
	ErrCustomerAlreadyExists = errors.New("customer already exists")
	ErrCustomerNotFound      = customerpublic.ErrCustomerNotFound
	ErrCustomerInactive      = customerpublic.ErrCustomerInactive
)
