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

func (repository *PostgresRepository) ExistsByInvoiceID(ctx context.Context, invoiceID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM payments
			WHERE invoice_id = $1
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
		INSERT INTO payments (id, invoice_id, status, gateway_reference, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).Exec(
		ctx,
		query,
		payment.ID(),
		payment.InvoiceID(),
		payment.Status(),
		payment.GatewayReference(),
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

func (repository *PostgresRepository) GetByID(ctx context.Context, paymentID string) (entities.Payment, error) {
	const query = `
		SELECT id, invoice_id, status, gateway_reference, created_at, updated_at
		FROM payments
		WHERE id = $1
	`

	var (
		id               string
		invoiceID        string
		status           string
		gatewayReference string
		createdAt        time.Time
		updatedAt        time.Time
	)

	err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).
		QueryRow(ctx, query, paymentID).
		Scan(&id, &invoiceID, &status, &gatewayReference, &createdAt, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return entities.Payment{}, entities.ErrInvalidPaymentID
	}
	if err != nil {
		return entities.Payment{}, fmt.Errorf("query payment by id: %w", err)
	}

	payment, err := entities.RehydratePayment(id, invoiceID, status, gatewayReference, createdAt, updatedAt)
	if err != nil {
		return entities.Payment{}, fmt.Errorf("rehydrate payment: %w", err)
	}

	return payment, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
