package entities

import "errors"

var (
	ErrInvalidCustomerID     = errors.New("invalid customer id")
	ErrCustomerNameRequired  = errors.New("customer name is required")
	ErrCustomerAlreadyExists = errors.New("customer already exists")
	ErrCustomerNotFound      = errors.New("customer not found")
	ErrCustomerInactive      = errors.New("customer is inactive")
)
