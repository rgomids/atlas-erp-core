package entities

import "errors"

var (
	ErrInvalidPaymentID        = errors.New("invalid payment id")
	ErrInvalidBillingReference = errors.New("invalid billing reference")
	ErrInvalidInvoiceReference = errors.New("invalid invoice reference")
	ErrPaymentAlreadyExists    = errors.New("payment already exists for invoice")
)
