package events

import "time"

type PaymentApproved struct {
	PaymentID        string
	BillingID        string
	InvoiceID        string
	GatewayReference string
	ApprovedAt       time.Time
}

func (PaymentApproved) Name() string {
	return "PaymentApproved"
}
