package event

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/rgomids/atlas-erp-core/internal/shared/correlation"
)

type Metadata struct {
	EventID       string    `json:"event_id"`
	EventName     string    `json:"event_name"`
	OccurredAt    time.Time `json:"occurred_at"`
	AggregateID   string    `json:"aggregate_id"`
	CorrelationID string    `json:"correlation_id"`
}

type Envelope[T any] struct {
	Metadata Metadata `json:"metadata"`
	Payload  T        `json:"payload"`
}

func (envelope Envelope[T]) EventMetadata() Metadata {
	return envelope.Metadata
}

func (envelope Envelope[T]) EventPayload() any {
	return envelope.Payload
}

type Descriptor struct {
	Name           string
	ProducerModule string
	Aggregate      string
	Description    string
}

func NewEnvelope[T any](ctx context.Context, eventName string, aggregateID string, occurredAt time.Time, payload T) Envelope[T] {
	return Envelope[T]{
		Metadata: NewMetadata(ctx, eventName, aggregateID, occurredAt),
		Payload:  payload,
	}
}

func NewMetadata(ctx context.Context, eventName string, aggregateID string, occurredAt time.Time) Metadata {
	correlationID := correlation.ID(ctx)
	if correlationID == "" {
		correlationID = uuid.NewString()
	}

	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	return Metadata{
		EventID:       uuid.NewString(),
		EventName:     eventName,
		OccurredAt:    occurredAt.UTC(),
		AggregateID:   aggregateID,
		CorrelationID: correlationID,
	}
}
