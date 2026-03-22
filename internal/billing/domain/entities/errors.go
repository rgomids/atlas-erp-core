package entities

import "errors"

var (
	ErrInvalidBillingID        = errors.New("invalid billing id")
	ErrInvalidInvoiceReference = errors.New("invalid invoice reference")
	ErrBillingAlreadyExists    = errors.New("billing already exists for invoice")
	ErrBillingNotFound         = errors.New("billing not found")
	ErrBillingAlreadyApproved  = errors.New("billing already approved")
)
