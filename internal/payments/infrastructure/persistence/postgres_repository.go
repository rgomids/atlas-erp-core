package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rgomids/atlas-erp-core/internal/payments/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/payments/domain/repositories"
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
)

var _ repositories.PaymentRepository = (*PostgresRepository)(nil)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (repository *PostgresRepository) HasApprovedByInvoiceID(ctx context.Context, invoiceID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM payments
			WHERE invoice_id = $1
			  AND status = 'Approved'
		)
	`

	var exists bool
	if err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).
		QueryRow(ctx, query, invoiceID).
		Scan(&exists); err != nil {
		return false, fmt.Errorf("query payment existence: %w", err)
	}

	return exists, nil
}

func (repository *PostgresRepository) Save(ctx context.Context, payment entities.Payment) error {
	const query = `
		INSERT INTO payments (id, billing_id, invoice_id, attempt_number, idempotency_key, status, gateway_reference, failure_category, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).Exec(
		ctx,
		query,
		payment.ID(),
		payment.BillingID(),
		payment.InvoiceID(),
		payment.AttemptNumber(),
		payment.IdempotencyKey(),
		payment.Status(),
		payment.GatewayReference(),
		nullableFailureCategory(payment.FailureCategory()),
		payment.CreatedAt(),
		payment.UpdatedAt(),
	)
	if isUniqueViolation(err) {
		return entities.ErrPaymentAlreadyExists
	}
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}

	return nil
}

func (repository *PostgresRepository) Update(ctx context.Context, payment entities.Payment) error {
	const query = `
		UPDATE payments
		SET status = $2,
			gateway_reference = $3,
			failure_category = $4,
			updated_at = $5
		WHERE id = $1
	`

	result, err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).Exec(
		ctx,
		query,
		payment.ID(),
		payment.Status(),
		payment.GatewayReference(),
		nullableFailureCategory(payment.FailureCategory()),
		payment.UpdatedAt(),
	)
	if isUniqueViolation(err) {
		return entities.ErrPaymentAlreadyExists
	}
	if err != nil {
		return fmt.Errorf("update payment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return entities.ErrPaymentNotFound
	}

	return nil
}

func (repository *PostgresRepository) GetByID(ctx context.Context, paymentID string) (entities.Payment, error) {
	const query = `
		SELECT id, billing_id, invoice_id, attempt_number, idempotency_key, status, gateway_reference, failure_category, created_at, updated_at
		FROM payments
		WHERE id = $1
	`

	var (
		id               string
		billingID        string
		invoiceID        string
		attemptNumber    int
		idempotencyKey   string
		status           string
		gatewayReference string
		failureCategory  *string
		createdAt        time.Time
		updatedAt        time.Time
	)

	err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).
		QueryRow(ctx, query, paymentID).
		Scan(&id, &billingID, &invoiceID, &attemptNumber, &idempotencyKey, &status, &gatewayReference, &failureCategory, &createdAt, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return entities.Payment{}, entities.ErrPaymentNotFound
	}
	if err != nil {
		return entities.Payment{}, fmt.Errorf("query payment by id: %w", err)
	}

	payment, err := entities.RehydratePayment(id, billingID, invoiceID, attemptNumber, idempotencyKey, status, gatewayReference, derefString(failureCategory), createdAt, updatedAt)
	if err != nil {
		return entities.Payment{}, fmt.Errorf("rehydrate payment: %w", err)
	}

	return payment, nil
}

func (repository *PostgresRepository) ListByInvoiceID(ctx context.Context, invoiceID string) ([]entities.Payment, error) {
	const query = `
		SELECT id, billing_id, invoice_id, attempt_number, idempotency_key, status, gateway_reference, failure_category, created_at, updated_at
		FROM payments
		WHERE invoice_id = $1
		ORDER BY attempt_number ASC, created_at ASC
	`

	rows, err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).Query(ctx, query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("query payments by invoice: %w", err)
	}
	defer rows.Close()

	var payments []entities.Payment
	for rows.Next() {
		var (
			id               string
			billingID        string
			scannedInvoiceID string
			attemptNumber    int
			idempotencyKey   string
			status           string
			gatewayReference string
			failureCategory  *string
			createdAt        time.Time
			updatedAt        time.Time
		)

		if err := rows.Scan(&id, &billingID, &scannedInvoiceID, &attemptNumber, &idempotencyKey, &status, &gatewayReference, &failureCategory, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan payment row: %w", err)
		}

		payment, err := entities.RehydratePayment(id, billingID, scannedInvoiceID, attemptNumber, idempotencyKey, status, gatewayReference, derefString(failureCategory), createdAt, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("rehydrate payment row: %w", err)
		}

		payments = append(payments, payment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate payment rows: %w", err)
	}

	return payments, nil
}

func (repository *PostgresRepository) GetByBillingIDAndAttempt(ctx context.Context, billingID string, attemptNumber int) (entities.Payment, error) {
	const query = `
		SELECT id, billing_id, invoice_id, attempt_number, idempotency_key, status, gateway_reference, failure_category, created_at, updated_at
		FROM payments
		WHERE billing_id = $1
		  AND attempt_number = $2
	`

	var (
		id               string
		scannedBillingID string
		invoiceID        string
		scannedAttempt   int
		idempotencyKey   string
		status           string
		gatewayReference string
		failureCategory  *string
		createdAt        time.Time
		updatedAt        time.Time
	)

	err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).
		QueryRow(ctx, query, billingID, attemptNumber).
		Scan(&id, &scannedBillingID, &invoiceID, &scannedAttempt, &idempotencyKey, &status, &gatewayReference, &failureCategory, &createdAt, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return entities.Payment{}, entities.ErrPaymentNotFound
	}
	if err != nil {
		return entities.Payment{}, fmt.Errorf("query payment by billing attempt: %w", err)
	}

	payment, err := entities.RehydratePayment(id, scannedBillingID, invoiceID, scannedAttempt, idempotencyKey, status, gatewayReference, derefString(failureCategory), createdAt, updatedAt)
	if err != nil {
		return entities.Payment{}, fmt.Errorf("rehydrate payment by billing attempt: %w", err)
	}

	return payment, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func nullableFailureCategory(value entities.FailureCategory) any {
	if value == "" {
		return nil
	}

	return string(value)
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
