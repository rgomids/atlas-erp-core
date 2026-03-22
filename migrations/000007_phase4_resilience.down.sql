DROP INDEX IF EXISTS outbox_events_event_name_idx;
DROP INDEX IF EXISTS outbox_events_status_occurred_at_idx;
DROP TABLE IF EXISTS outbox_events;

DROP INDEX IF EXISTS payments_idempotency_key_uq;
DROP INDEX IF EXISTS payments_billing_attempt_uq;

ALTER TABLE payments
    DROP COLUMN IF EXISTS failure_category,
    DROP COLUMN IF EXISTS idempotency_key,
    DROP COLUMN IF EXISTS attempt_number;

ALTER TABLE billings
    DROP COLUMN IF EXISTS attempt_number,
    DROP COLUMN IF EXISTS customer_id;
