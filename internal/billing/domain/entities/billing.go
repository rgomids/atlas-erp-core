package entities

import (
	"strings"
	"time"
)

type Status string

const (
	StatusRequested Status = "Requested"
	StatusFailed    Status = "Failed"
	StatusApproved  Status = "Approved"
)

type Billing struct {
	id            string
	invoiceID     string
	customerID    string
	amountCents   int64
	dueDate       time.Time
	status        Status
	attemptNumber int
	createdAt     time.Time
	updatedAt     time.Time
}

func NewBilling(id, invoiceID, customerID string, amountCents int64, dueDate time.Time, now time.Time) (Billing, error) {
	billing := Billing{
		id:            strings.TrimSpace(id),
		invoiceID:     strings.TrimSpace(invoiceID),
		customerID:    strings.TrimSpace(customerID),
		amountCents:   amountCents,
		dueDate:       normalizeDate(dueDate),
		status:        StatusRequested,
		attemptNumber: 1,
		createdAt:     now.UTC(),
		updatedAt:     now.UTC(),
	}

	if err := billing.validate(); err != nil {
		return Billing{}, err
	}

	return billing, nil
}

func RehydrateBilling(
	id string,
	invoiceID string,
	customerID string,
	amountCents int64,
	dueDate time.Time,
	status string,
	attemptNumber int,
	createdAt time.Time,
	updatedAt time.Time,
) (Billing, error) {
	billing := Billing{
		id:            strings.TrimSpace(id),
		invoiceID:     strings.TrimSpace(invoiceID),
		customerID:    strings.TrimSpace(customerID),
		amountCents:   amountCents,
		dueDate:       normalizeDate(dueDate),
		status:        Status(strings.TrimSpace(status)),
		attemptNumber: attemptNumber,
		createdAt:     createdAt.UTC(),
		updatedAt:     updatedAt.UTC(),
	}

	if err := billing.validate(); err != nil {
		return Billing{}, err
	}

	return billing, nil
}

func (billing *Billing) MarkApproved(now time.Time) {
	if billing.status == StatusApproved {
		return
	}

	billing.status = StatusApproved
	billing.updatedAt = now.UTC()
}

func (billing *Billing) MarkFailed(now time.Time) {
	if billing.status == StatusFailed {
		return
	}

	billing.status = StatusFailed
	billing.updatedAt = now.UTC()
}

func (billing *Billing) MarkRequested(now time.Time) error {
	if billing.status == StatusApproved {
		return ErrBillingAlreadyApproved
	}
	if billing.status == StatusRequested {
		return nil
	}

	billing.status = StatusRequested
	billing.attemptNumber++
	billing.updatedAt = now.UTC()

	return nil
}

func (billing Billing) validate() error {
	if billing.id == "" {
		return ErrInvalidBillingID
	}
	if billing.invoiceID == "" {
		return ErrInvalidInvoiceReference
	}
	if billing.customerID == "" {
		return ErrInvalidCustomerReference
	}
	if billing.amountCents <= 0 {
		return ErrInvalidInvoiceReference
	}
	if billing.dueDate.IsZero() {
		return ErrInvalidInvoiceReference
	}
	if billing.attemptNumber <= 0 {
		return ErrInvalidAttemptNumber
	}
	if billing.status != StatusRequested && billing.status != StatusFailed && billing.status != StatusApproved {
		return ErrInvalidBillingID
	}

	return nil
}

func (billing Billing) ID() string {
	return billing.id
}

func (billing Billing) InvoiceID() string {
	return billing.invoiceID
}

func (billing Billing) CustomerID() string {
	return billing.customerID
}

func (billing Billing) AmountCents() int64 {
	return billing.amountCents
}

func (billing Billing) DueDate() time.Time {
	return billing.dueDate
}

func (billing Billing) Status() Status {
	return billing.status
}

func (billing Billing) AttemptNumber() int {
	return billing.attemptNumber
}

func (billing Billing) CreatedAt() time.Time {
	return billing.createdAt
}

func (billing Billing) UpdatedAt() time.Time {
	return billing.updatedAt
}

func normalizeDate(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}

	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
