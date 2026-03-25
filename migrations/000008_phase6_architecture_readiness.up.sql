ALTER TABLE outbox_events
    RENAME COLUMN request_id TO correlation_id;

ALTER TABLE outbox_events
    ADD COLUMN aggregate_id TEXT NULL,
    ADD COLUMN processed_at TIMESTAMPTZ NULL,
    ADD COLUMN failed_at TIMESTAMPTZ NULL,
    ADD COLUMN error_message TEXT NULL;

UPDATE outbox_events
SET status = LOWER(status),
    aggregate_id = COALESCE(
        payload->'metadata'->>'aggregate_id',
        payload->>'PaymentID',
        payload->>'BillingID',
        payload->>'InvoiceID',
        payload->>'CustomerID',
        id::text
    );

ALTER TABLE outbox_events
    DROP CONSTRAINT IF EXISTS outbox_events_status_check;

ALTER TABLE outbox_events
    ALTER COLUMN aggregate_id SET NOT NULL;

ALTER TABLE outbox_events
    ADD CONSTRAINT outbox_events_status_check CHECK (status IN ('pending', 'processed', 'failed'));

CREATE INDEX IF NOT EXISTS outbox_events_aggregate_id_idx ON outbox_events (aggregate_id);
