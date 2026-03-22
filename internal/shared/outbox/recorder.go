package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
)

const statusPending = "Pending"

type PostgresRecorder struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewPostgresRecorder(pool *pgxpool.Pool) *PostgresRecorder {
	return &PostgresRecorder{
		pool: pool,
		now:  time.Now,
	}
}

func (recorder *PostgresRecorder) Record(ctx context.Context, record sharedevent.EventRecord) error {
	const query = `
		INSERT INTO outbox_events (id, event_name, emitter_module, request_id, status, payload, occurred_at, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8)
	`

	_, err := sharedpostgres.ExecutorFromContext(ctx, recorder.pool).Exec(
		ctx,
		query,
		uuid.NewString(),
		record.EventName,
		record.EmitterModule,
		record.RequestID,
		statusPending,
		record.Payload,
		record.OccurredAt,
		recorder.now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}

	return nil
}
