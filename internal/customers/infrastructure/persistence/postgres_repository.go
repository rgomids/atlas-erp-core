package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rgomids/atlas-erp-core/internal/customers/domain/entities"
	"github.com/rgomids/atlas-erp-core/internal/customers/domain/repositories"
	"github.com/rgomids/atlas-erp-core/internal/customers/domain/valueobjects"
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
)

var _ repositories.CustomerRepository = (*PostgresRepository)(nil)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (repository *PostgresRepository) ExistsByDocument(ctx context.Context, document string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM customers
			WHERE document = $1
		)
	`

	var exists bool
	if err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).
		QueryRow(ctx, query, document).
		Scan(&exists); err != nil {
		return false, fmt.Errorf("query customer document existence: %w", err)
	}

	return exists, nil
}

func (repository *PostgresRepository) Save(ctx context.Context, customer entities.Customer) error {
	const query = `
		INSERT INTO customers (id, name, document, email, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).Exec(
		ctx,
		query,
		customer.ID(),
		customer.Name(),
		customer.Document().Value(),
		customer.Email().Value(),
		customer.Status(),
		customer.CreatedAt(),
		customer.UpdatedAt(),
	)
	if isUniqueViolation(err) {
		return entities.ErrCustomerAlreadyExists
	}
	if err != nil {
		return fmt.Errorf("insert customer: %w", err)
	}

	return nil
}

func (repository *PostgresRepository) GetByID(ctx context.Context, customerID string) (entities.Customer, error) {
	const query = `
		SELECT id, name, document, email, status, created_at, updated_at
		FROM customers
		WHERE id = $1
	`

	var (
		id        string
		name      string
		document  string
		email     string
		status    string
		createdAt time.Time
		updatedAt time.Time
	)

	err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).
		QueryRow(ctx, query, customerID).
		Scan(&id, &name, &document, &email, &status, &createdAt, &updatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return entities.Customer{}, entities.ErrCustomerNotFound
	}
	if err != nil {
		return entities.Customer{}, fmt.Errorf("query customer by id: %w", err)
	}

	doc, err := valueobjects.NewDocument(document)
	if err != nil {
		return entities.Customer{}, fmt.Errorf("rehydrate customer document: %w", err)
	}

	addr, err := valueobjects.NewEmail(email)
	if err != nil {
		return entities.Customer{}, fmt.Errorf("rehydrate customer email: %w", err)
	}

	customer, err := entities.RehydrateCustomer(id, name, doc, addr, status, createdAt, updatedAt)
	if err != nil {
		return entities.Customer{}, fmt.Errorf("rehydrate customer: %w", err)
	}

	return customer, nil
}

func (repository *PostgresRepository) Update(ctx context.Context, customer entities.Customer) error {
	const query = `
		UPDATE customers
		SET name = $2,
			email = $3,
			status = $4,
			updated_at = $5
		WHERE id = $1
	`

	result, err := sharedpostgres.ExecutorFromContext(ctx, repository.pool).Exec(
		ctx,
		query,
		customer.ID(),
		customer.Name(),
		customer.Email().Value(),
		customer.Status(),
		customer.UpdatedAt(),
	)
	if err != nil {
		return fmt.Errorf("update customer: %w", err)
	}

	if result.RowsAffected() == 0 {
		return entities.ErrCustomerNotFound
	}

	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
