package entities

import (
	"errors"
	"testing"
	"time"

	"github.com/rgomids/atlas-erp-core/internal/customers/domain/valueobjects"
)

func TestNewCustomerNormalizesDocumentAndEmail(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	customer, err := NewCustomer("customer-id", "Atlas Co", "123.456.789-00", "  TEAM@ATLAS.IO  ", now)
	if err != nil {
		t.Fatalf("expected customer to be created, got error: %v", err)
	}

	if customer.Document().Value() != "12345678900" {
		t.Fatalf("expected normalized document, got %q", customer.Document().Value())
	}

	if customer.Email().Value() != "team@atlas.io" {
		t.Fatalf("expected normalized email, got %q", customer.Email().Value())
	}

	if customer.Status() != StatusActive {
		t.Fatalf("expected active status, got %q", customer.Status())
	}
}

func TestNewCustomerRejectsInvalidValueObjects(t *testing.T) {
	t.Parallel()

	_, err := NewCustomer("customer-id", "Atlas Co", "123", "invalid", time.Now())
	if !errors.Is(err, valueobjects.ErrInvalidDocument) {
		t.Fatalf("expected invalid document error, got %v", err)
	}
}

func TestCustomerUpdateAndDeactivate(t *testing.T) {
	t.Parallel()

	customer, err := NewCustomer("customer-id", "Atlas Co", "12345678900", "team@atlas.io", time.Now())
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	if err := customer.UpdateProfile("Atlas Updated", "billing@atlas.io", time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("update customer: %v", err)
	}

	if customer.Name() != "Atlas Updated" {
		t.Fatalf("expected updated name, got %q", customer.Name())
	}

	customer.Deactivate(time.Now().Add(2 * time.Minute))
	if customer.Status() != StatusInactive {
		t.Fatalf("expected inactive customer, got %q", customer.Status())
	}
}
