package entities

import (
	"strings"
	"time"
)

type Status string

const (
	StatusPending  Status = "Pending"
	StatusApproved Status = "Approved"
	StatusFailed   Status = "Failed"
)

type Payment struct {
	id               string
	invoiceID        string
	status           Status
	gatewayReference string
	createdAt        time.Time
	updatedAt        time.Time
}

func NewPayment(id, invoiceID string, now time.Time) (Payment, error) {
	payment := Payment{
		id:        strings.TrimSpace(id),
		invoiceID: strings.TrimSpace(invoiceID),
		status:    StatusPending,
		createdAt: now.UTC(),
		updatedAt: now.UTC(),
	}

	if err := payment.validate(); err != nil {
		return Payment{}, err
	}

	return payment, nil
}

func RehydratePayment(
	id string,
	invoiceID string,
	status string,
	gatewayReference string,
	createdAt time.Time,
	updatedAt time.Time,
) (Payment, error) {
	payment := Payment{
		id:               strings.TrimSpace(id),
		invoiceID:        strings.TrimSpace(invoiceID),
		status:           Status(strings.TrimSpace(status)),
		gatewayReference: strings.TrimSpace(gatewayReference),
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
	payment.updatedAt = now.UTC()
}

func (payment *Payment) MarkFailed(gatewayReference string, now time.Time) {
	payment.status = StatusFailed
	payment.gatewayReference = strings.TrimSpace(gatewayReference)
	payment.updatedAt = now.UTC()
}

func (payment Payment) validate() error {
	if payment.id == "" {
		return ErrInvalidPaymentID
	}
	if payment.invoiceID == "" {
		return ErrInvalidInvoiceReference
	}
	if payment.status != StatusPending && payment.status != StatusApproved && payment.status != StatusFailed {
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

func (payment Payment) Status() Status {
	return payment.status
}

func (payment Payment) GatewayReference() string {
	return payment.gatewayReference
}

func (payment Payment) CreatedAt() time.Time {
	return payment.createdAt
}

func (payment Payment) UpdatedAt() time.Time {
	return payment.updatedAt
}
