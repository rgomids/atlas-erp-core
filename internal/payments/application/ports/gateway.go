package ports

import (
	"context"
	"time"
)

type GatewayRequest struct {
	BillingID   string
	InvoiceID   string
	AmountCents int64
	DueDate     time.Time
}

type GatewayResult struct {
	Status           string
	GatewayReference string
}

type PaymentGateway interface {
	Process(ctx context.Context, request GatewayRequest) (GatewayResult, error)
}
