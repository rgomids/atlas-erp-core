package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txContextKey string

const transactionKey txContextKey = "postgres_tx"

type QueryExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) TxManager {
	return TxManager{pool: pool}
}

func (manager TxManager) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	tx, err := manager.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	txContext := context.WithValue(ctx, transactionKey, tx)
	if err := fn(txContext); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return fmt.Errorf("rollback transaction: %w", rollbackErr)
		}

		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func ExecutorFromContext(ctx context.Context, fallback *pgxpool.Pool) QueryExecutor {
	tx, ok := ctx.Value(transactionKey).(pgx.Tx)
	if ok {
		return tx
	}

	return fallback
}
