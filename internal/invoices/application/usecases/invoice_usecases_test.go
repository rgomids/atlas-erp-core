package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	customerentities "github.com/rgomids/atlas-erp-core/internal/customers/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/repositories"
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
	createInvoice := NewCreateInvoice(repository, customerCheckerFake{})
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

			createInvoice := NewCreateInvoice(newInvoiceRepositoryFake(), customerCheckerFake{})

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
	createInvoice := NewCreateInvoice(repository, customerCheckerFake{err: customerentities.ErrCustomerNotFound})

	_, err := createInvoice.Execute(context.Background(), CreateInvoiceInput{
		CustomerID:  "2af9b675-4c54-4b1e-9e1f-e56028421b6d",
		AmountCents: 1500,
		DueDate:     "2026-03-25",
	})
	if !errors.Is(err, customerentities.ErrCustomerNotFound) {
		t.Fatalf("expected customer not found error, got %v", err)
	}

	createInvoice = NewCreateInvoice(repository, customerCheckerFake{})
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
