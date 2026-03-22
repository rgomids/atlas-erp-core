package dto

import "time"

type Payment struct {
	ID               string    `json:"id"`
	InvoiceID        string    `json:"invoice_id"`
	Status           string    `json:"status"`
	AttemptNumber    int       `json:"attempt_number"`
	FailureCategory  string    `json:"failure_category,omitempty"`
	GatewayReference string    `json:"gateway_reference"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
