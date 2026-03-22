DROP INDEX IF EXISTS payments_invoice_id_approved_uq;

CREATE UNIQUE INDEX payments_invoice_id_uq ON payments (invoice_id);

ALTER TABLE payments
    DROP COLUMN IF EXISTS billing_id;
