package entities

import (
	"strings"
	"time"
)

type Status string

const (
	StatusPending   Status = "Pending"
	StatusPaid      Status = "Paid"
	StatusOverdue   Status = "Overdue"
	StatusCancelled Status = "Cancelled"
)

type Invoice struct {
	id          string
	customerID  string
	amountCents int64
	dueDate     time.Time
	status      Status
	createdAt   time.Time
	updatedAt   time.Time
	paidAt      *time.Time
}

func NewInvoice(id, customerID string, amountCents int64, dueDate time.Time, now time.Time) (Invoice, error) {
	invoice := Invoice{
		id:          strings.TrimSpace(id),
		customerID:  strings.TrimSpace(customerID),
		amountCents: amountCents,
		dueDate:     normalizeDate(dueDate),
		status:      StatusPending,
		createdAt:   now.UTC(),
		updatedAt:   now.UTC(),
	}

	if err := invoice.validate(); err != nil {
		return Invoice{}, err
	}

	return invoice, nil
}

func RehydrateInvoice(
	id string,
	customerID string,
	amountCents int64,
	dueDate time.Time,
	status string,
	createdAt time.Time,
	updatedAt time.Time,
	paidAt *time.Time,
) (Invoice, error) {
	invoice := Invoice{
		id:          strings.TrimSpace(id),
		customerID:  strings.TrimSpace(customerID),
		amountCents: amountCents,
		dueDate:     normalizeDate(dueDate),
		status:      Status(strings.TrimSpace(status)),
		createdAt:   createdAt.UTC(),
		updatedAt:   updatedAt.UTC(),
		paidAt:      normalizeOptionalTime(paidAt),
	}

	if err := invoice.validate(); err != nil {
		return Invoice{}, err
	}

	return invoice, nil
}

func (invoice *Invoice) MarkPaid(now time.Time) error {
	if invoice.status == StatusPaid {
		return ErrInvoiceImmutable
	}
	if invoice.status == StatusCancelled {
		return ErrInvoiceNotPayable
	}

	paidAt := now.UTC()
	invoice.status = StatusPaid
	invoice.paidAt = &paidAt
	invoice.updatedAt = paidAt

	return nil
}

func (invoice Invoice) validate() error {
	if invoice.id == "" {
		return ErrInvalidInvoiceID
	}
	if invoice.customerID == "" {
		return ErrInvalidCustomerReference
	}
	if invoice.amountCents <= 0 {
		return ErrInvoiceAmountMustBePositive
	}
	if invoice.dueDate.IsZero() {
		return ErrInvoiceDueDateRequired
	}
	if invoice.status != StatusPending &&
		invoice.status != StatusPaid &&
		invoice.status != StatusOverdue &&
		invoice.status != StatusCancelled {
		return ErrInvoiceNotPayable
	}

	return nil
}

func (invoice Invoice) ID() string {
	return invoice.id
}

func (invoice Invoice) CustomerID() string {
	return invoice.customerID
}

func (invoice Invoice) AmountCents() int64 {
	return invoice.amountCents
}

func (invoice Invoice) DueDate() time.Time {
	return invoice.dueDate
}

func (invoice Invoice) Status() Status {
	return invoice.status
}

func (invoice Invoice) CreatedAt() time.Time {
	return invoice.createdAt
}

func (invoice Invoice) UpdatedAt() time.Time {
	return invoice.updatedAt
}

func (invoice Invoice) PaidAt() *time.Time {
	return invoice.paidAt
}

func (invoice Invoice) IsPayable() bool {
	return invoice.status == StatusPending || invoice.status == StatusOverdue
}

func normalizeDate(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}

	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func normalizeOptionalTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	normalized := value.UTC()
	return &normalized
}
