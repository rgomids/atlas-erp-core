package events

import "time"

type BillingRequested struct {
	BillingID   string
	InvoiceID   string
	AmountCents int64
	DueDate     time.Time
}

func (BillingRequested) Name() string {
	return "BillingRequested"
}
