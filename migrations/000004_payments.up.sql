CREATE TABLE payments (
    id UUID PRIMARY KEY,
    invoice_id UUID NOT NULL REFERENCES invoices (id),
    status TEXT NOT NULL CHECK (status IN ('Pending', 'Approved', 'Failed')),
    gateway_reference TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX payments_invoice_id_uq ON payments (invoice_id);
