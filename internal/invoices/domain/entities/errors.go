package entities

import "errors"

var (
	ErrInvalidInvoiceID            = errors.New("invalid invoice id")
	ErrInvalidCustomerReference    = errors.New("invalid customer reference")
	ErrInvoiceAmountMustBePositive = errors.New("invoice amount must be greater than zero")
	ErrInvoiceDueDateRequired      = errors.New("invoice due date is required")
	ErrInvoiceNotFound             = errors.New("invoice not found")
	ErrInvoiceImmutable            = errors.New("invoice cannot be changed after payment")
	ErrInvoiceNotPayable           = errors.New("invoice is not payable")
)
