DROP INDEX IF EXISTS outbox_events_aggregate_id_idx;

ALTER TABLE outbox_events
    DROP CONSTRAINT IF EXISTS outbox_events_status_check;

UPDATE outbox_events
SET status = INITCAP(status);

ALTER TABLE outbox_events
    ADD CONSTRAINT outbox_events_status_check CHECK (status IN ('Pending', 'Processed', 'Failed'));

ALTER TABLE outbox_events
    DROP COLUMN IF EXISTS error_message,
    DROP COLUMN IF EXISTS failed_at,
    DROP COLUMN IF EXISTS processed_at,
    DROP COLUMN IF EXISTS aggregate_id;

ALTER TABLE outbox_events
    RENAME COLUMN correlation_id TO request_id;
