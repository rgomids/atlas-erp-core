package usecases

import "github.com/rgomids/atlas-erp-core/internal/billing/application/ports"
import "github.com/rgomids/atlas-erp-core/internal/billing/domain/entities"

func toSnapshot(billing entities.Billing) ports.BillingSnapshot {
	return ports.BillingSnapshot{
		ID:            billing.ID(),
		InvoiceID:     billing.InvoiceID(),
		CustomerID:    billing.CustomerID(),
		AmountCents:   billing.AmountCents(),
		DueDate:       billing.DueDate(),
		Status:        string(billing.Status()),
		AttemptNumber: billing.AttemptNumber(),
	}
}
