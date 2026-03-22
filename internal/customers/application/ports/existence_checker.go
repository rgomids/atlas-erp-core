package ports

import "context"

type ExistenceChecker interface {
	ExistsActiveCustomer(ctx context.Context, customerID string) error
}
