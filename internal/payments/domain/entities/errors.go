package entities

import "errors"

var (
	ErrInvalidPaymentID        = errors.New("invalid payment id")
	ErrInvalidInvoiceReference = errors.New("invalid invoice reference")
	ErrPaymentAlreadyExists    = errors.New("payment already exists for invoice")
)
