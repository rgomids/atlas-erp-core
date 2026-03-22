package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rgomids/atlas-erp-core/internal/customers/domain/entities"
	customerevents "github.com/rgomids/atlas-erp-core/internal/customers/domain/events"
	"github.com/rgomids/atlas-erp-core/internal/customers/domain/repositories"
	"github.com/rgomids/atlas-erp-core/internal/customers/domain/valueobjects"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
)

type customerRepositoryFake struct {
	byID       map[string]entities.Customer
	byDocument map[string]string
}

var _ repositories.CustomerRepository = (*customerRepositoryFake)(nil)

func newCustomerRepositoryFake() *customerRepositoryFake {
	return &customerRepositoryFake{
		byID:       map[string]entities.Customer{},
		byDocument: map[string]string{},
	}
}

func (repository *customerRepositoryFake) ExistsByDocument(_ context.Context, document string) (bool, error) {
	_, exists := repository.byDocument[document]
	return exists, nil
}

func (repository *customerRepositoryFake) Save(_ context.Context, customer entities.Customer) error {
	repository.byID[customer.ID()] = customer
	repository.byDocument[customer.Document().Value()] = customer.ID()
	return nil
}

func (repository *customerRepositoryFake) GetByID(_ context.Context, customerID string) (entities.Customer, error) {
	customer, exists := repository.byID[customerID]
	if !exists {
		return entities.Customer{}, entities.ErrCustomerNotFound
	}

	return customer, nil
}

func (repository *customerRepositoryFake) Update(_ context.Context, customer entities.Customer) error {
	if _, exists := repository.byID[customer.ID()]; !exists {
		return entities.ErrCustomerNotFound
	}

	repository.byID[customer.ID()] = customer
	return nil
}

func TestCreateCustomerRejectsDuplicateDocument(t *testing.T) {
	t.Parallel()

	repository := newCustomerRepositoryFake()
	createCustomer := NewCreateCustomer(repository, nil)
	createCustomer.now = func() time.Time { return time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC) }

	firstCustomer, err := createCustomer.Execute(context.Background(), CreateCustomerInput{
		Name:     "Atlas Co",
		Document: "12345678900",
		Email:    "team@atlas.io",
	})
	if err != nil {
		t.Fatalf("create first customer: %v", err)
	}

	if firstCustomer.Status != "Active" {
		t.Fatalf("expected active customer, got %q", firstCustomer.Status)
	}

	_, err = createCustomer.Execute(context.Background(), CreateCustomerInput{
		Name:     "Atlas Co 2",
		Document: "123.456.789-00",
		Email:    "ops@atlas.io",
	})
	if !errors.Is(err, entities.ErrCustomerAlreadyExists) {
		t.Fatalf("expected duplicate customer error, got %v", err)
	}
}

func TestCreateCustomerCreatesActiveCustomer(t *testing.T) {
	t.Parallel()

	repository := newCustomerRepositoryFake()
	createCustomer := NewCreateCustomer(repository, nil)
	createCustomer.now = func() time.Time { return time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC) }

	customer, err := createCustomer.Execute(context.Background(), CreateCustomerInput{
		Name:     "Atlas Co",
		Document: "12345678900",
		Email:    "team@atlas.io",
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	if customer.ID == "" {
		t.Fatal("expected customer id to be generated")
	}

	if customer.Status != "Active" {
		t.Fatalf("expected active customer, got %q", customer.Status)
	}
}

func TestCreateCustomerRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		input       CreateCustomerInput
		expectedErr error
	}{
		{
			name: "blank name",
			input: CreateCustomerInput{
				Name:     "   ",
				Document: "12345678900",
				Email:    "team@atlas.io",
			},
			expectedErr: entities.ErrCustomerNameRequired,
		},
		{
			name: "invalid email",
			input: CreateCustomerInput{
				Name:     "Atlas Co",
				Document: "12345678900",
				Email:    "invalid",
			},
			expectedErr: valueobjects.ErrInvalidEmail,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			createCustomer := NewCreateCustomer(newCustomerRepositoryFake(), nil)

			_, err := createCustomer.Execute(context.Background(), testCase.input)
			if !errors.Is(err, testCase.expectedErr) {
				t.Fatalf("expected error %v, got %v", testCase.expectedErr, err)
			}
		})
	}
}

func TestUpdateAndDeactivateCustomer(t *testing.T) {
	t.Parallel()

	repository := newCustomerRepositoryFake()
	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	createCustomer := NewCreateCustomer(repository, nil)
	createCustomer.now = func() time.Time { return now }

	customer, err := createCustomer.Execute(context.Background(), CreateCustomerInput{
		Name:     "Atlas Co",
		Document: "12345678900",
		Email:    "team@atlas.io",
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	updateCustomer := NewUpdateCustomer(repository)
	updateCustomer.now = func() time.Time { return now.Add(time.Minute) }

	updatedCustomer, err := updateCustomer.Execute(context.Background(), UpdateCustomerInput{
		ID:    customer.ID,
		Name:  "Atlas Updated",
		Email: "billing@atlas.io",
	})
	if err != nil {
		t.Fatalf("update customer: %v", err)
	}

	if updatedCustomer.Name != "Atlas Updated" {
		t.Fatalf("expected updated name, got %q", updatedCustomer.Name)
	}

	deactivateCustomer := NewDeactivateCustomer(repository)
	deactivateCustomer.now = func() time.Time { return now.Add(2 * time.Minute) }

	deactivatedCustomer, err := deactivateCustomer.Execute(context.Background(), DeactivateCustomerInput{ID: customer.ID})
	if err != nil {
		t.Fatalf("deactivate customer: %v", err)
	}

	if deactivatedCustomer.Status != "Inactive" {
		t.Fatalf("expected inactive customer, got %q", deactivatedCustomer.Status)
	}
}

func TestUpdateCustomerRejectsInvalidCustomerID(t *testing.T) {
	t.Parallel()

	updateCustomer := NewUpdateCustomer(newCustomerRepositoryFake())

	_, err := updateCustomer.Execute(context.Background(), UpdateCustomerInput{
		ID:    "not-a-uuid",
		Name:  "Atlas Updated",
		Email: "billing@atlas.io",
	})
	if !errors.Is(err, entities.ErrInvalidCustomerID) {
		t.Fatalf("expected invalid customer id error, got %v", err)
	}
}

func TestCreateCustomerPublishesCustomerCreatedEvent(t *testing.T) {
	t.Parallel()

	repository := newCustomerRepositoryFake()
	bus := sharedevent.NewSyncBus()
	var createdEvents []customerevents.CustomerCreated

	sharedevent.Subscribe(bus, customerevents.CustomerCreated{}.Name(), "test", sharedevent.HandlerFunc(func(_ context.Context, event sharedevent.Event) error {
		createdEvents = append(createdEvents, event.(customerevents.CustomerCreated))
		return nil
	}))

	createCustomer := NewCreateCustomer(repository, bus)
	createCustomer.now = func() time.Time { return time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC) }

	customer, err := createCustomer.Execute(context.Background(), CreateCustomerInput{
		Name:     "Atlas Co",
		Document: "12345678900",
		Email:    "team@atlas.io",
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	if len(createdEvents) != 1 {
		t.Fatalf("expected 1 customer created event, got %d", len(createdEvents))
	}

	if createdEvents[0].CustomerID != customer.ID {
		t.Fatalf("expected event customer id %q, got %q", customer.ID, createdEvents[0].CustomerID)
	}
}
