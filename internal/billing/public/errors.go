package public

import "errors"

var (
	ErrBillingNotFound        = errors.New("billing not found")
	ErrBillingAlreadyApproved = errors.New("billing already approved")
)
