CREATE TABLE billings (
    id UUID PRIMARY KEY,
    invoice_id UUID NOT NULL REFERENCES invoices (id),
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    due_date DATE NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('Requested', 'Failed', 'Approved')),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX billings_invoice_id_uq ON billings (invoice_id);
