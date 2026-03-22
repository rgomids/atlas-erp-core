ALTER TABLE billings
    ADD COLUMN customer_id UUID NULL,
    ADD COLUMN attempt_number INTEGER NOT NULL DEFAULT 1 CHECK (attempt_number > 0);

UPDATE billings
SET customer_id = invoices.customer_id
FROM invoices
WHERE invoices.id = billings.invoice_id
  AND billings.customer_id IS NULL;

ALTER TABLE billings
    ALTER COLUMN customer_id SET NOT NULL;

ALTER TABLE payments
    ADD COLUMN attempt_number INTEGER NOT NULL DEFAULT 1 CHECK (attempt_number > 0),
    ADD COLUMN idempotency_key TEXT NULL,
    ADD COLUMN failure_category TEXT NULL CHECK (failure_category IN ('gateway_declined', 'gateway_timeout', 'gateway_error'));

UPDATE payments
SET idempotency_key = CONCAT('billing:', billing_id::text, ':attempt:', attempt_number::text)
WHERE idempotency_key IS NULL;

ALTER TABLE payments
    ALTER COLUMN idempotency_key SET NOT NULL;

CREATE UNIQUE INDEX payments_billing_attempt_uq ON payments (billing_id, attempt_number);
CREATE UNIQUE INDEX payments_idempotency_key_uq ON payments (idempotency_key);

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY,
    event_name TEXT NOT NULL,
    emitter_module TEXT NOT NULL,
    request_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('Pending', 'Processed', 'Failed')),
    payload JSONB NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX outbox_events_status_occurred_at_idx ON outbox_events (status, occurred_at);
CREATE INDEX outbox_events_event_name_idx ON outbox_events (event_name);
