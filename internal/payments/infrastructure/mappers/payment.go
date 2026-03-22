package mappers

import (
	"github.com/rgomids/atlas-erp-core/internal/payments/application/dto"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
)

func ToPaymentDTO(payment entities.Payment) dto.Payment {
	return dto.Payment{
		ID:               payment.ID(),
		InvoiceID:        payment.InvoiceID(),
		Status:           string(payment.Status()),
		GatewayReference: payment.GatewayReference(),
		CreatedAt:        payment.CreatedAt(),
		UpdatedAt:        payment.UpdatedAt(),
	}
}
