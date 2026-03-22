package entities

import "errors"

var (
	ErrInvalidPaymentID         = errors.New("invalid payment id")
	ErrInvalidBillingReference  = errors.New("invalid billing reference")
	ErrInvalidInvoiceReference  = errors.New("invalid invoice reference")
	ErrInvalidCustomerReference = errors.New("invalid customer reference")
	ErrInvalidAttemptNumber     = errors.New("invalid attempt number")
	ErrInvalidIdempotencyKey    = errors.New("invalid idempotency key")
	ErrPaymentAlreadyExists     = errors.New("payment already exists for invoice")
	ErrPaymentNotFound          = errors.New("payment not found")
)
