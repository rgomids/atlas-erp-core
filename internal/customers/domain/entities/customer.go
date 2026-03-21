package entities

import (
	"strings"
	"time"

	"github.com/rgomids/atlas-erp-core/internal/customers/domain/valueobjects"
)

type Status string

const (
	StatusActive   Status = "Active"
	StatusInactive Status = "Inactive"
)

type Customer struct {
	id        string
	name      string
	document  valueobjects.Document
	email     valueobjects.Email
	status    Status
	createdAt time.Time
	updatedAt time.Time
}

func NewCustomer(id, name, rawDocument, rawEmail string, now time.Time) (Customer, error) {
	if strings.TrimSpace(id) == "" {
		return Customer{}, ErrInvalidCustomerID
	}

	document, err := valueobjects.NewDocument(rawDocument)
	if err != nil {
		return Customer{}, err
	}

	email, err := valueobjects.NewEmail(rawEmail)
	if err != nil {
		return Customer{}, err
	}

	customer := Customer{
		id:        id,
		name:      strings.TrimSpace(name),
		document:  document,
		email:     email,
		status:    StatusActive,
		createdAt: now.UTC(),
		updatedAt: now.UTC(),
	}

	if err := customer.validate(); err != nil {
		return Customer{}, err
	}

	return customer, nil
}

func RehydrateCustomer(
	id string,
	name string,
	document valueobjects.Document,
	email valueobjects.Email,
	status string,
	createdAt time.Time,
	updatedAt time.Time,
) (Customer, error) {
	customer := Customer{
		id:        strings.TrimSpace(id),
		name:      strings.TrimSpace(name),
		document:  document,
		email:     email,
		status:    Status(strings.TrimSpace(status)),
		createdAt: createdAt.UTC(),
		updatedAt: updatedAt.UTC(),
	}

	if err := customer.validate(); err != nil {
		return Customer{}, err
	}

	return customer, nil
}

func (customer *Customer) UpdateProfile(name, rawEmail string, now time.Time) error {
	email, err := valueobjects.NewEmail(rawEmail)
	if err != nil {
		return err
	}

	customer.name = strings.TrimSpace(name)
	customer.email = email
	customer.updatedAt = now.UTC()

	return customer.validate()
}

func (customer *Customer) Deactivate(now time.Time) {
	customer.status = StatusInactive
	customer.updatedAt = now.UTC()
}

func (customer Customer) validate() error {
	if customer.id == "" {
		return ErrInvalidCustomerID
	}

	if customer.name == "" {
		return ErrCustomerNameRequired
	}

	if customer.status != StatusActive && customer.status != StatusInactive {
		return ErrCustomerInactive
	}

	return nil
}

func (customer Customer) ID() string {
	return customer.id
}

func (customer Customer) Name() string {
	return customer.name
}

func (customer Customer) Document() valueobjects.Document {
	return customer.document
}

func (customer Customer) Email() valueobjects.Email {
	return customer.email
}

func (customer Customer) Status() Status {
	return customer.status
}

func (customer Customer) CreatedAt() time.Time {
	return customer.createdAt
}

func (customer Customer) UpdatedAt() time.Time {
	return customer.updatedAt
}

func (customer Customer) IsActive() bool {
	return customer.status == StatusActive
}
