package entities

import (
	"errors"

	billingpublic "github.com/rgomids/atlas-erp-core/internal/billing/public"
)

var (
	ErrInvalidBillingID         = errors.New("invalid billing id")
	ErrInvalidInvoiceReference  = errors.New("invalid invoice reference")
	ErrInvalidCustomerReference = errors.New("invalid customer reference")
	ErrInvalidAttemptNumber     = errors.New("invalid attempt number")
	ErrBillingAlreadyExists     = errors.New("billing already exists for invoice")
	ErrBillingNotFound          = billingpublic.ErrBillingNotFound
	ErrBillingAlreadyApproved   = billingpublic.ErrBillingAlreadyApproved
)
