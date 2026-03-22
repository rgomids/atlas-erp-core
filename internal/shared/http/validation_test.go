package httpapi

import "testing"

func TestRequireNonBlank(t *testing.T) {
	t.Parallel()

	if err := RequireNonBlank("document", "123"); err != nil {
		t.Fatalf("expected non blank value to pass, got %v", err)
	}

	err := RequireNonBlank("document", "   ")
	if err == nil || err.Error() != "document is required" {
		t.Fatalf("expected required field error, got %v", err)
	}
}

func TestRequireUUID(t *testing.T) {
	t.Parallel()

	validID := "1adf3d42-7b1d-4d2b-a7d6-5d977b7576fe"
	if err := RequireUUID("customer_id", validID); err != nil {
		t.Fatalf("expected valid uuid to pass, got %v", err)
	}

	err := RequireUUID("customer_id", "invalid")
	if err == nil || err.Error() != "customer_id must be a valid UUID" {
		t.Fatalf("expected invalid uuid error, got %v", err)
	}
}

func TestRequirePositiveInt64(t *testing.T) {
	t.Parallel()

	if err := RequirePositiveInt64("amount_cents", 1); err != nil {
		t.Fatalf("expected positive amount to pass, got %v", err)
	}

	err := RequirePositiveInt64("amount_cents", 0)
	if err == nil || err.Error() != "amount_cents must be greater than zero" {
		t.Fatalf("expected positive amount error, got %v", err)
	}
}

func TestRequireDate(t *testing.T) {
	t.Parallel()

	if err := RequireDate("due_date", "2026-03-25"); err != nil {
		t.Fatalf("expected valid date to pass, got %v", err)
	}

	err := RequireDate("due_date", "25/03/2026")
	if err == nil || err.Error() != "due_date must use YYYY-MM-DD" {
		t.Fatalf("expected date format error, got %v", err)
	}
}
