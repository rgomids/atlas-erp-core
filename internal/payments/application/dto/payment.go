package dto

import "time"

type Payment struct {
	ID               string    `json:"id"`
	InvoiceID        string    `json:"invoice_id"`
	Status           string    `json:"status"`
	GatewayReference string    `json:"gateway_reference"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
