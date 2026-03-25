package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	sharedevent "github.com/rgomids/atlas-erp-core/internal/shared/event"
	sharedpostgres "github.com/rgomids/atlas-erp-core/internal/shared/postgres"
)

const (
	statusPending   = "pending"
	statusProcessed = "processed"
	statusFailed    = "failed"
)

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

func (recorder *PostgresRecorder) Append(ctx context.Context, record sharedevent.EventRecord) error {
	const query = `
		INSERT INTO outbox_events (id, event_name, aggregate_id, emitter_module, correlation_id, status, payload, occurred_at, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9)
	`

	_, err := sharedpostgres.ExecutorFromContext(ctx, recorder.pool).Exec(
		ctx,
		query,
		record.EventID,
		record.EventName,
		record.AggregateID,
		record.EmitterModule,
		record.CorrelationID,
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

func (recorder *PostgresRecorder) MarkProcessed(ctx context.Context, eventID string, processedAt time.Time) error {
	const query = `
		UPDATE outbox_events
		SET status = $2,
		    processed_at = $3,
		    failed_at = NULL,
		    error_message = NULL
		WHERE id = $1
	`

	_, err := sharedpostgres.ExecutorFromContext(ctx, recorder.pool).Exec(
		ctx,
		query,
		eventID,
		statusProcessed,
		processedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("mark outbox event processed: %w", err)
	}

	return nil
}

func (recorder *PostgresRecorder) MarkFailed(ctx context.Context, eventID string, failedAt time.Time, errorMessage string) error {
	const query = `
		UPDATE outbox_events
		SET status = $2,
		    failed_at = $3,
		    error_message = $4
		WHERE id = $1
	`

	_, err := sharedpostgres.ExecutorFromContext(ctx, recorder.pool).Exec(
		ctx,
		query,
		eventID,
		statusFailed,
		failedAt.UTC(),
		errorMessage,
	)
	if err != nil {
		return fmt.Errorf("mark outbox event failed: %w", err)
	}

	return nil
}
