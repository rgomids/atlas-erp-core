package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/invoices/domain/repositories"
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
)

var _ repositories.InvoiceRepository = (*PostgresRepository)(nil)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (repository *PostgresRepository) Save(ctx context.Context, invoice entities.Invoice) error {
	const query = `
		INSERT INTO invoices (id, customer_id, amount_cents, due_date, status, created_at, updated_at, paid_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).Exec(
		ctx,
		query,
		invoice.ID(),
		invoice.CustomerID(),
		invoice.AmountCents(),
		invoice.DueDate(),
		invoice.Status(),
		invoice.CreatedAt(),
		invoice.UpdatedAt(),
		invoice.PaidAt(),
	)
	if err != nil {
		return fmt.Errorf("insert invoice: %w", err)
	}

	return nil
}

func (repository *PostgresRepository) GetByID(ctx context.Context, invoiceID string) (entities.Invoice, error) {
	const query = `
		SELECT id, customer_id, amount_cents, due_date, status, created_at, updated_at, paid_at
		FROM invoices
		WHERE id = $1
	`

	var (
		id          string
		customerID  string
		amountCents int64
		dueDate     time.Time
		status      string
		createdAt   time.Time
		updatedAt   time.Time
		paidAt      *time.Time
	)

	err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).
		QueryRow(ctx, query, invoiceID).
		Scan(&id, &customerID, &amountCents, &dueDate, &status, &createdAt, &updatedAt, &paidAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return entities.Invoice{}, entities.ErrInvoiceNotFound
	}
	if err != nil {
		return entities.Invoice{}, fmt.Errorf("query invoice by id: %w", err)
	}

	invoice, err := entities.RehydrateInvoice(id, customerID, amountCents, dueDate, status, createdAt, updatedAt, paidAt)
	if err != nil {
		return entities.Invoice{}, fmt.Errorf("rehydrate invoice: %w", err)
	}

	return invoice, nil
}

func (repository *PostgresRepository) ListByCustomerID(ctx context.Context, customerID string) ([]entities.Invoice, error) {
	const query = `
		SELECT id, customer_id, amount_cents, due_date, status, created_at, updated_at, paid_at
		FROM invoices
		WHERE customer_id = $1
		ORDER BY created_at DESC
	`

	rows, err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).Query(ctx, query, customerID)
	if err != nil {
		return nil, fmt.Errorf("query invoices by customer: %w", err)
	}
	defer rows.Close()

	var invoices []entities.Invoice
	for rows.Next() {
		var (
			id                string
			scannedCustomerID string
			amountCents       int64
			dueDate           time.Time
			status            string
			createdAt         time.Time
			updatedAt         time.Time
			paidAt            *time.Time
		)

		if err := rows.Scan(&id, &scannedCustomerID, &amountCents, &dueDate, &status, &createdAt, &updatedAt, &paidAt); err != nil {
			return nil, fmt.Errorf("scan invoice row: %w", err)
		}

		invoice, err := entities.RehydrateInvoice(id, scannedCustomerID, amountCents, dueDate, status, createdAt, updatedAt, paidAt)
		if err != nil {
			return nil, fmt.Errorf("rehydrate invoice row: %w", err)
		}

		invoices = append(invoices, invoice)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invoice rows: %w", err)
	}

	return invoices, nil
}

func (repository *PostgresRepository) Update(ctx context.Context, invoice entities.Invoice) error {
	const query = `
		UPDATE invoices
		SET status = $2,
			updated_at = $3,
			paid_at = $4
		WHERE id = $1
	`

	result, err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).Exec(
		ctx,
		query,
		invoice.ID(),
		invoice.Status(),
		invoice.UpdatedAt(),
		invoice.PaidAt(),
	)
	if err != nil {
		return fmt.Errorf("update invoice: %w", err)
	}

	if result.RowsAffected() == 0 {
		return entities.ErrInvoiceNotFound
	}

	return nil
}
