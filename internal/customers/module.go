package customers

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rgomids/atlas-erp-core/internal/customers/application/ports"
	"github.com/rgomids/atlas-erp-core/internal/customers/application/usecases"
	"github.com/rgomids/atlas-erp-core/internal/customers/domain/entities"
	customershttp "github.com/rgomids/atlas-erp-core/internal/customers/infrastructure/http"
	"github.com/rgomids/atlas-erp-core/internal/customers/infrastructure/persistence"
)

type Module struct {
	handler customershttp.Handler
	checker ports.ExistenceChecker
}

func NewModule(pool *pgxpool.Pool) Module {
	repository := persistence.NewPostgresRepository(pool)

	return Module{
		handler: customershttp.NewHandler(
			usecases.NewCreateCustomer(repository),
			usecases.NewUpdateCustomer(repository),
			usecases.NewDeactivateCustomer(repository),
		),
		checker: existenceChecker{repository: repository},
	}
}

func (module Module) Routes(router chi.Router) {
	module.handler.Routes(router)
}

func (module Module) ExistenceChecker() ports.ExistenceChecker {
	return module.checker
}

type existenceChecker struct {
	repository *persistence.PostgresRepository
}

func (checker existenceChecker) ExistsActiveCustomer(ctx context.Context, customerID string) error {
	customer, err := checker.repository.GetByID(ctx, customerID)
	if err != nil {
		return err
	}

	if !customer.IsActive() {
		return entities.ErrCustomerInactive
	}

	return nil
}
