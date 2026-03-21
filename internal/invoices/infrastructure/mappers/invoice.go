package mappers

import (
	"github.com/rgomids/atlas-erp-core/internal/invoices/application/dto"
	"github.com/rgomids/atlas-erp-core/internal/invoices/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/entities"
)

func ToInvoiceDTO(invoice entities.Invoice) dto.Invoice {
	return dto.Invoice{
		ID:          invoice.ID(),
		CustomerID:  invoice.CustomerID(),
		AmountCents: invoice.AmountCents(),
		DueDate:     invoice.DueDate().Format("2006-01-02"),
		Status:      string(invoice.Status()),
		CreatedAt:   invoice.CreatedAt(),
		UpdatedAt:   invoice.UpdatedAt(),
		PaidAt:      invoice.PaidAt(),
	}
}

func ToInvoiceDTOs(invoices []entities.Invoice) []dto.Invoice {
	items := make([]dto.Invoice, 0, len(invoices))
	for _, invoice := range invoices {
		items = append(items, ToInvoiceDTO(invoice))
	}

	return items
}

func ToSnapshot(invoice entities.Invoice) ports.InvoiceSnapshot {
	return ports.InvoiceSnapshot{
		ID:          invoice.ID(),
		CustomerID:  invoice.CustomerID(),
		AmountCents: invoice.AmountCents(),
		DueDate:     invoice.DueDate(),
		Status:      string(invoice.Status()),
	}
}
