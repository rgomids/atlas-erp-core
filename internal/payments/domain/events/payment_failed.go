package events

import "time"

type PaymentFailed struct {
	PaymentID        string
	BillingID        string
	InvoiceID        string
	CustomerID       string
	AttemptNumber    int
	IdempotencyKey   string
	FailureCategory  string
	GatewayReference string
	FailedAt         time.Time
}

func (PaymentFailed) Name() string {
	return "PaymentFailed"
}
