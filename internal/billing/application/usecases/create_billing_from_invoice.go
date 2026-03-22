package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/rgomids/atlas-erp-core/internal/billing/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/billing/domain/entities"
	billingevents "github.com/rgomids/atlas-erp-core/internal/billing/domain/events"
	"github.com/rgomids/atlas-erp-core/internal/billing/domain/repositories"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

type CreateBillingFromInvoiceInput struct {
	InvoiceID   string
	CustomerID  string
	AmountCents int64
	DueDate     time.Time
}

type CreateBillingFromInvoice struct {
	repository repositories.BillingRepository
	bus        sharedevent.EventBus
	now        func() time.Time
}

func NewCreateBillingFromInvoice(repository repositories.BillingRepository, bus sharedevent.EventBus) CreateBillingFromInvoice {
	return CreateBillingFromInvoice{
		repository: repository,
		bus:        bus,
		now:        time.Now,
	}
}

func (usecase CreateBillingFromInvoice) Execute(ctx context.Context, input CreateBillingFromInvoiceInput) (ports.BillingSnapshot, error) {
	if _, err := uuid.Parse(input.InvoiceID); err != nil {
		return ports.BillingSnapshot{}, entities.ErrInvalidInvoiceReference
	}
	if _, err := uuid.Parse(input.CustomerID); err != nil {
		return ports.BillingSnapshot{}, entities.ErrInvalidCustomerReference
	}

	existing, err := usecase.repository.GetByInvoiceID(ctx, input.InvoiceID)
	switch {
	case err == nil:
		return toSnapshot(existing), nil
	case !errors.Is(err, entities.ErrBillingNotFound):
		return ports.BillingSnapshot{}, err
	}

	billing, err := entities.NewBilling(uuid.NewString(), input.InvoiceID, input.CustomerID, input.AmountCents, input.DueDate, usecase.now())
	if err != nil {
		return ports.BillingSnapshot{}, err
	}

	if err := usecase.repository.Save(ctx, billing); err != nil {
		if errors.Is(err, entities.ErrBillingAlreadyExists) {
			existing, getErr := usecase.repository.GetByInvoiceID(ctx, input.InvoiceID)
			if getErr != nil {
				return ports.BillingSnapshot{}, getErr
			}

			return toSnapshot(existing), nil
		}

		return ports.BillingSnapshot{}, fmt.Errorf("save billing: %w", err)
	}

	if err := sharedevent.Publish(ctx, usecase.bus, "billing", billingevents.BillingRequested{
		BillingID:     billing.ID(),
		InvoiceID:     billing.InvoiceID(),
		CustomerID:    billing.CustomerID(),
		AmountCents:   billing.AmountCents(),
		DueDate:       billing.DueDate(),
		AttemptNumber: billing.AttemptNumber(),
	}); err != nil {
		return ports.BillingSnapshot{}, err
	}

	return toSnapshot(billing), nil
}
