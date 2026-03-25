package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	customerpublic "github.com/rgomids/atlas-erp-core/internal/customers/public"
	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/repositories"
	invoiceevents "github.com/rgomids/atlas-erp-core/internal/invoices/public/events"
	paymentevents "github.com/rgomids/atlas-erp-core/internal/payments/public/events"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

type invoiceRepositoryFake struct {
	byID       map[string]entities.Invoice
	byCustomer map[string][]string
}

var _ repositories.InvoiceRepository = (*invoiceRepositoryFake)(nil)

func newInvoiceRepositoryFake() *invoiceRepositoryFake {
	return &invoiceRepositoryFake{
		byID:       map[string]entities.Invoice{},
		byCustomer: map[string][]string{},
	}
}

func (repository *invoiceRepositoryFake) Save(_ context.Context, invoice entities.Invoice) error {
	repository.byID[invoice.ID()] = invoice
	repository.byCustomer[invoice.CustomerID()] = append(repository.byCustomer[invoice.CustomerID()], invoice.ID())
	return nil
}

func (repository *invoiceRepositoryFake) GetByID(_ context.Context, invoiceID string) (entities.Invoice, error) {
	invoice, exists := repository.byID[invoiceID]
	if !exists {
		return entities.Invoice{}, entities.ErrInvoiceNotFound
	}

	return invoice, nil
}

func (repository *invoiceRepositoryFake) ListByCustomerID(_ context.Context, customerID string) ([]entities.Invoice, error) {
	var invoices []entities.Invoice
	for _, invoiceID := range repository.byCustomer[customerID] {
		invoices = append(invoices, repository.byID[invoiceID])
	}

	return invoices, nil
}

func (repository *invoiceRepositoryFake) Update(_ context.Context, invoice entities.Invoice) error {
	repository.byID[invoice.ID()] = invoice
	return nil
}

type customerCheckerFake struct {
	err error
}

func (checker customerCheckerFake) ExistsActiveCustomer(context.Context, string) error {
	return checker.err
}

func TestCreateInvoiceRequiresExistingCustomer(t *testing.T) {
	t.Parallel()

	repository := newInvoiceRepositoryFake()
	createInvoice := NewCreateInvoice(repository, customerCheckerFake{}, nil)
	createInvoice.now = func() time.Time { return time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC) }

	invoice, err := createInvoice.Execute(context.Background(), CreateInvoiceInput{
		CustomerID:  "2af9b675-4c54-4b1e-9e1f-e56028421b6d",
		AmountCents: 1500,
		DueDate:     "2026-03-25",
	})
	if err != nil {
		t.Fatalf("expected invoice to be created, got error: %v", err)
	}

	if invoice.Status != "Pending" {
		t.Fatalf("expected pending invoice, got %q", invoice.Status)
	}
}

func TestCreateInvoiceRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		input       CreateInvoiceInput
		expectedErr error
	}{
		{
			name: "invalid customer id",
			input: CreateInvoiceInput{
				CustomerID:  "invalid",
				AmountCents: 1500,
				DueDate:     "2026-03-25",
			},
			expectedErr: entities.ErrInvalidCustomerReference,
		},
		{
			name: "invalid amount",
			input: CreateInvoiceInput{
				CustomerID:  "2af9b675-4c54-4b1e-9e1f-e56028421b6d",
				AmountCents: 0,
				DueDate:     "2026-03-25",
			},
			expectedErr: entities.ErrInvoiceAmountMustBePositive,
		},
		{
			name: "invalid due date format",
			input: CreateInvoiceInput{
				CustomerID:  "2af9b675-4c54-4b1e-9e1f-e56028421b6d",
				AmountCents: 1500,
				DueDate:     "25/03/2026",
			},
			expectedErr: entities.ErrInvoiceDueDateRequired,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			createInvoice := NewCreateInvoice(newInvoiceRepositoryFake(), customerCheckerFake{}, nil)

			_, err := createInvoice.Execute(context.Background(), testCase.input)
			if !errors.Is(err, testCase.expectedErr) {
				t.Fatalf("expected error %v, got %v", testCase.expectedErr, err)
			}
		})
	}
}

func TestCreateInvoicePropagatesCustomerErrorsAndListsInvoices(t *testing.T) {
	t.Parallel()

	repository := newInvoiceRepositoryFake()
	createInvoice := NewCreateInvoice(repository, customerCheckerFake{err: customerpublic.ErrCustomerNotFound}, nil)

	_, err := createInvoice.Execute(context.Background(), CreateInvoiceInput{
		CustomerID:  "2af9b675-4c54-4b1e-9e1f-e56028421b6d",
		AmountCents: 1500,
		DueDate:     "2026-03-25",
	})
	if !errors.Is(err, customerpublic.ErrCustomerNotFound) {
		t.Fatalf("expected customer not found error, got %v", err)
	}

	createInvoice = NewCreateInvoice(repository, customerCheckerFake{}, nil)
	createInvoice.now = func() time.Time { return time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC) }

	customerID := "2af9b675-4c54-4b1e-9e1f-e56028421b6d"
	if _, err := createInvoice.Execute(context.Background(), CreateInvoiceInput{
		CustomerID:  customerID,
		AmountCents: 1500,
		DueDate:     "2026-03-25",
	}); err != nil {
		t.Fatalf("create first invoice: %v", err)
	}
	if _, err := createInvoice.Execute(context.Background(), CreateInvoiceInput{
		CustomerID:  customerID,
		AmountCents: 3200,
		DueDate:     "2026-03-26",
	}); err != nil {
		t.Fatalf("create second invoice: %v", err)
	}

	listInvoices := NewListCustomerInvoices(repository)
	invoices, err := listInvoices.Execute(context.Background(), ListCustomerInvoicesInput{CustomerID: customerID})
	if err != nil {
		t.Fatalf("list invoices: %v", err)
	}

	if len(invoices) != 2 {
		t.Fatalf("expected 2 invoices, got %d", len(invoices))
	}
}

func TestListCustomerInvoicesRejectsInvalidCustomerID(t *testing.T) {
	t.Parallel()

	listInvoices := NewListCustomerInvoices(newInvoiceRepositoryFake())

	_, err := listInvoices.Execute(context.Background(), ListCustomerInvoicesInput{CustomerID: "invalid"})
	if !errors.Is(err, entities.ErrInvalidCustomerReference) {
		t.Fatalf("expected invalid customer reference error, got %v", err)
	}
}

func TestCreateInvoicePublishesInvoiceCreatedEvent(t *testing.T) {
	t.Parallel()

	repository := newInvoiceRepositoryFake()
	bus := sharedevent.NewSyncBus()
	var createdEvents []invoiceevents.InvoiceCreated

	sharedevent.Subscribe(bus, invoiceevents.EventNameInvoiceCreated, "test", sharedevent.HandlerFunc(func(_ context.Context, event sharedevent.Event) error {
		createdEvents = append(createdEvents, event.(invoiceevents.InvoiceCreated))
		return nil
	}))

	createInvoice := NewCreateInvoice(repository, customerCheckerFake{}, bus)
	createInvoice.now = func() time.Time { return time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC) }

	invoice, err := createInvoice.Execute(context.Background(), CreateInvoiceInput{
		CustomerID:  "2af9b675-4c54-4b1e-9e1f-e56028421b6d",
		AmountCents: 1500,
		DueDate:     "2026-03-25",
	})
	if err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	if len(createdEvents) != 1 {
		t.Fatalf("expected 1 invoice created event, got %d", len(createdEvents))
	}

	if createdEvents[0].Payload.InvoiceID != invoice.ID {
		t.Fatalf("expected invoice id %q, got %q", invoice.ID, createdEvents[0].Payload.InvoiceID)
	}
}

func TestApplyPaymentApprovedMarksInvoicePaidAndPublishesInvoicePaid(t *testing.T) {
	t.Parallel()

	repository := newInvoiceRepositoryFake()
	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	invoice, err := entities.NewInvoice(
		"1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe",
		"2af9b675-4c54-4b1e-9e1f-e56028421b6d",
		1500,
		time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		now,
	)
	if err != nil {
		t.Fatalf("new invoice: %v", err)
	}
	if err := repository.Save(context.Background(), invoice); err != nil {
		t.Fatalf("save invoice: %v", err)
	}

	bus := sharedevent.NewSyncBus()
	var paidEvents []invoiceevents.InvoicePaid
	sharedevent.Subscribe(bus, invoiceevents.EventNameInvoicePaid, "test", sharedevent.HandlerFunc(func(_ context.Context, event sharedevent.Event) error {
		paidEvents = append(paidEvents, event.(invoiceevents.InvoicePaid))
		return nil
	}))

	applyPaymentApproved := NewApplyPaymentApproved(repository, bus)

	err = applyPaymentApproved.Execute(
		context.Background(),
		paymentevents.NewPaymentApproved(
			context.Background(),
			"payment-id",
			"billing-id",
			invoice.ID(),
			"customer-id",
			1,
			"billing:billing-id:attempt:1",
			"gw-001",
			now.Add(time.Minute),
		),
	)
	if err != nil {
		t.Fatalf("apply payment approved: %v", err)
	}

	storedInvoice, err := repository.GetByID(context.Background(), invoice.ID())
	if err != nil {
		t.Fatalf("get invoice: %v", err)
	}

	if storedInvoice.Status() != entities.StatusPaid {
		t.Fatalf("expected paid invoice, got %q", storedInvoice.Status())
	}

	if len(paidEvents) != 1 {
		t.Fatalf("expected 1 invoice paid event, got %d", len(paidEvents))
	}

	if paidEvents[0].EventMetadata().AggregateID != invoice.ID() {
		t.Fatalf("expected invoice paid aggregate id %q, got %q", invoice.ID(), paidEvents[0].EventMetadata().AggregateID)
	}
}
