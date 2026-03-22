package events

import "time"

type BillingRequested struct {
	BillingID     string
	InvoiceID     string
	CustomerID    string
	AmountCents   int64
	DueDate       time.Time
	AttemptNumber int
}

func (BillingRequested) Name() string {
	return "BillingRequested"
}
