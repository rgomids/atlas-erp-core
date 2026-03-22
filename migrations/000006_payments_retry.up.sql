ALTER TABLE payments
    ADD COLUMN billing_id UUID NULL REFERENCES billings (id);

INSERT INTO billings (id, invoice_id, amount_cents, due_date, status, created_at, updated_at)
SELECT
    payments.id,
    invoices.id,
    invoices.amount_cents,
    invoices.due_date,
    CASE
        WHEN payments.status = 'Approved' THEN 'Approved'
        ELSE 'Failed'
    END,
    payments.created_at,
    payments.updated_at
FROM payments
INNER JOIN invoices ON invoices.id = payments.invoice_id
ON CONFLICT (invoice_id) DO NOTHING;

UPDATE payments
SET billing_id = payments.id
WHERE billing_id IS NULL;

ALTER TABLE payments
    ALTER COLUMN billing_id SET NOT NULL;

DROP INDEX IF EXISTS payments_invoice_id_uq;

CREATE UNIQUE INDEX payments_invoice_id_approved_uq ON payments (invoice_id)
WHERE status = 'Approved';
