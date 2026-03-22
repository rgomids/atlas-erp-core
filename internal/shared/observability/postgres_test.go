package observability

import "testing"

func TestSanitizeSQLUsesOperationAndTableOnly(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		statement         string
		expectedOperation string
		expectedTable     string
	}{
		{
			name:              "insert",
			statement:         "INSERT INTO invoices (id, customer_id) VALUES ($1, $2)",
			expectedOperation: "insert",
			expectedTable:     "invoices",
		},
		{
			name:              "select",
			statement:         "SELECT id, status FROM payments WHERE invoice_id = $1",
			expectedOperation: "select",
			expectedTable:     "payments",
		},
		{
			name:              "update",
			statement:         "UPDATE billings SET status = $1 WHERE id = $2",
			expectedOperation: "update",
			expectedTable:     "billings",
		},
		{
			name:              "delete",
			statement:         "DELETE FROM outbox_events WHERE id = $1",
			expectedOperation: "delete",
			expectedTable:     "outbox_events",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			operation, table := sanitizeSQL(testCase.statement)
			if operation != testCase.expectedOperation {
				t.Fatalf("expected operation %q, got %q", testCase.expectedOperation, operation)
			}

			if table != testCase.expectedTable {
				t.Fatalf("expected table %q, got %q", testCase.expectedTable, table)
			}
		})
	}
}
