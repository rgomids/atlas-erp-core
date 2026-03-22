package events

import "time"

type PaymentApproved struct {
	PaymentID        string
	BillingID        string
	InvoiceID        string
	CustomerID       string
	AttemptNumber    int
	IdempotencyKey   string
	GatewayReference string
	ApprovedAt       time.Time
}

func (PaymentApproved) Name() string {
	return "PaymentApproved"
}
