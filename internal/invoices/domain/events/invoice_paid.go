package events

import "time"

type InvoicePaid struct {
	InvoiceID string
	PaidAt    time.Time
}

func (InvoicePaid) Name() string {
	return "InvoicePaid"
}
