package customers

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rgomids/atlas-erp-core/internal/customers/application/usecases"
	"github.com/rgomids/atlas-erp-core/internal/customers/domain/entities"
	customershttp "github.com/rgomids/atlas-erp-core/internal/customers/infrastructure/http"
	"github.com/rgomids/atlas-erp-core/internal/customers/infrastructure/persistence"
	customerpublic "github.com/rgomids/atlas-erp-core/internal/customers/public"
	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	"github.com/rgomids/atlas-erp-core/internal/shared/observability"
)

type Module struct {
	handler customershttp.Handler
	checker customerpublic.ExistenceChecker
}

func NewModule(pool *pgxpool.Pool, bus sharedevent.EventBus, telemetry ...*observability.Runtime) Module {
	repository := persistence.NewPostgresRepository(pool)
	obs := observability.FromOptional(telemetry...)

	return Module{
		handler: customershttp.NewHandler(
			usecases.NewCreateCustomer(repository, bus, obs),
			usecases.NewUpdateCustomer(repository, obs),
			usecases.NewDeactivateCustomer(repository, obs),
		),
		checker: existenceChecker{repository: repository},
	}
}

func (module Module) Routes(router chi.Router) {
	module.handler.Routes(router)
}

func (module Module) ExistenceChecker() customerpublic.ExistenceChecker {
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
