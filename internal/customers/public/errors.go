package public

import "errors"

var (
	ErrCustomerNotFound = errors.New("customer not found")
	ErrCustomerInactive = errors.New("customer is inactive")
)
