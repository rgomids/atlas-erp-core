package events

import "time"

type InvoiceCreated struct {
	InvoiceID   string
	CustomerID  string
	AmountCents int64
	DueDate     time.Time
	CreatedAt   time.Time
}

func (InvoiceCreated) Name() string {
	return "InvoiceCreated"
}
