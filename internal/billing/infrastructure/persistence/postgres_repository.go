package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rgomids/atlas-erp-core/internal/billing/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/billing/domain/repositories"
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
)

var _ repositories.BillingRepository = (*PostgresRepository)(nil)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (repository *PostgresRepository) Save(ctx context.Context, billing entities.Billing) error {
	const query = `
		INSERT INTO billings (id, invoice_id, amount_cents, due_date, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).Exec(
		ctx,
		query,
		billing.ID(),
		billing.InvoiceID(),
		billing.AmountCents(),
		billing.DueDate(),
		billing.Status(),
		billing.CreatedAt(),
		billing.UpdatedAt(),
	)
	if isUniqueViolation(err) {
		return entities.ErrBillingAlreadyExists
	}
	if err != nil {
		return fmt.Errorf("insert billing: %w", err)
	}

	return nil
}

func (repository *PostgresRepository) GetByID(ctx context.Context, billingID string) (entities.Billing, error) {
	const query = `
		SELECT id, invoice_id, amount_cents, due_date, status, created_at, updated_at
		FROM billings
		WHERE id = $1
	`

	return repository.getOne(ctx, query, billingID)
}

func (repository *PostgresRepository) GetByInvoiceID(ctx context.Context, invoiceID string) (entities.Billing, error) {
	const query = `
		SELECT id, invoice_id, amount_cents, due_date, status, created_at, updated_at
		FROM billings
		WHERE invoice_id = $1
	`

	return repository.getOne(ctx, query, invoiceID)
}

func (repository *PostgresRepository) Update(ctx context.Context, billing entities.Billing) error {
	const query = `
		UPDATE billings
		SET status = $2,
			updated_at = $3
		WHERE id = $1
	`

	result, err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).Exec(
		ctx,
		query,
		billing.ID(),
		billing.Status(),
		billing.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("update billing: %w", err)
	}

	if result.RowsAffected() == 0 {
		return entities.ErrBillingNotFound
	}

	return nil
}

func (repository *PostgresRepository) getOne(ctx context.Context, query string, argument string) (entities.Billing, error) {
	var (
		id          string
		invoiceID   string
		amountCents int64
		dueDate     time.Time
		status      string
		createdAt   time.Time
		updatedAt   time.Time
	)

	err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).
		QueryRow(ctx, query, argument).
		Scan(&id, &invoiceID, &amountCents, &dueDate, &status, &createdAt, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return entities.Billing{}, entities.ErrBillingNotFound
	}
	if err != nil {
		return entities.Billing{}, fmt.Errorf("query billing: %w", err)
	}

	billing, err := entities.RehydrateBilling(id, invoiceID, amountCents, dueDate, status, createdAt, updatedAt)
	if err != nil {
		return entities.Billing{}, fmt.Errorf("rehydrate billing: %w", err)
	}

	return billing, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
