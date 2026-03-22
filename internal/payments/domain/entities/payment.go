package entities

import (
	"strings"
	"time"
)

type Status string
type FailureCategory string

const (
	StatusPending  Status = "Pending"
	StatusApproved Status = "Approved"
	StatusFailed   Status = "Failed"

	FailureCategoryGatewayDeclined FailureCategory = "gateway_declined"
	FailureCategoryGatewayTimeout  FailureCategory = "gateway_timeout"
	FailureCategoryGatewayError    FailureCategory = "gateway_error"
)

type Payment struct {
	id               string
	billingID        string
	invoiceID        string
	attemptNumber    int
	idempotencyKey   string
	status           Status
	gatewayReference string
	failureCategory  FailureCategory
	createdAt        time.Time
	updatedAt        time.Time
}

func NewPayment(id, billingID, invoiceID string, attemptNumber int, idempotencyKey string, now time.Time) (Payment, error) {
	payment := Payment{
		id:             strings.TrimSpace(id),
		billingID:      strings.TrimSpace(billingID),
		invoiceID:      strings.TrimSpace(invoiceID),
		attemptNumber:  attemptNumber,
		idempotencyKey: strings.TrimSpace(idempotencyKey),
		status:         StatusPending,
		createdAt:      now.UTC(),
		updatedAt:      now.UTC(),
	}

	if err := payment.validate(); err != nil {
		return Payment{}, err
	}

	return payment, nil
}

func RehydratePayment(
	id string,
	billingID string,
	invoiceID string,
	attemptNumber int,
	idempotencyKey string,
	status string,
	gatewayReference string,
	failureCategory string,
	createdAt time.Time,
	updatedAt time.Time,
) (Payment, error) {
	payment := Payment{
		id:               strings.TrimSpace(id),
		billingID:        strings.TrimSpace(billingID),
		invoiceID:        strings.TrimSpace(invoiceID),
		attemptNumber:    attemptNumber,
		idempotencyKey:   strings.TrimSpace(idempotencyKey),
		status:           Status(strings.TrimSpace(status)),
		gatewayReference: strings.TrimSpace(gatewayReference),
		failureCategory:  FailureCategory(strings.TrimSpace(failureCategory)),
		createdAt:        createdAt.UTC(),
		updatedAt:        updatedAt.UTC(),
	}

	if err := payment.validate(); err != nil {
		return Payment{}, err
	}

	return payment, nil
}

func (payment *Payment) MarkApproved(gatewayReference string, now time.Time) {
	payment.status = StatusApproved
	payment.gatewayReference = strings.TrimSpace(gatewayReference)
	payment.failureCategory = ""
	payment.updatedAt = now.UTC()
}

func (payment *Payment) MarkFailed(gatewayReference string, failureCategory FailureCategory, now time.Time) {
	payment.status = StatusFailed
	payment.gatewayReference = strings.TrimSpace(gatewayReference)
	payment.failureCategory = failureCategory
	payment.updatedAt = now.UTC()
}

func (payment Payment) validate() error {
	if payment.id == "" {
		return ErrInvalidPaymentID
	}
	if payment.billingID == "" {
		return ErrInvalidBillingReference
	}
	if payment.invoiceID == "" {
		return ErrInvalidInvoiceReference
	}
	if payment.attemptNumber <= 0 {
		return ErrInvalidAttemptNumber
	}
	if payment.idempotencyKey == "" {
		return ErrInvalidIdempotencyKey
	}
	if payment.status != StatusPending && payment.status != StatusApproved && payment.status != StatusFailed {
		return ErrInvalidPaymentID
	}
	if payment.status == StatusFailed &&
		payment.failureCategory != FailureCategoryGatewayDeclined &&
		payment.failureCategory != FailureCategoryGatewayTimeout &&
		payment.failureCategory != FailureCategoryGatewayError {
		return ErrInvalidPaymentID
	}
	if payment.status != StatusFailed && payment.failureCategory != "" {
		return ErrInvalidPaymentID
	}

	return nil
}

func (payment Payment) ID() string {
	return payment.id
}

func (payment Payment) InvoiceID() string {
	return payment.invoiceID
}

func (payment Payment) BillingID() string {
	return payment.billingID
}

func (payment Payment) Status() Status {
	return payment.status
}

func (payment Payment) AttemptNumber() int {
	return payment.attemptNumber
}

func (payment Payment) IdempotencyKey() string {
	return payment.idempotencyKey
}

func (payment Payment) GatewayReference() string {
	return payment.gatewayReference
}

func (payment Payment) FailureCategory() FailureCategory {
	return payment.failureCategory
}

func (payment Payment) CreatedAt() time.Time {
	return payment.createdAt
}

func (payment Payment) UpdatedAt() time.Time {
	return payment.updatedAt
}
