package ports

import (
	"context"

	invoiceports "github.com/rgomids/atlas-erp-core/internal/invoices/application/ports"
)

type GatewayRequest struct {
	Invoice invoiceports.InvoiceSnapshot
}

type GatewayResult struct {
	Status           string
	GatewayReference string
}

type PaymentGateway interface {
	Process(ctx context.Context, request GatewayRequest) (GatewayResult, error)
}
