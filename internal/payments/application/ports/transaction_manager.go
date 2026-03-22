package ports

import "context"

type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(context.Context) error) error
}
