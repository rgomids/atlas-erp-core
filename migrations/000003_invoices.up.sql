CREATE TABLE invoices (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES customers (id),
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    due_date DATE NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('Pending', 'Paid', 'Overdue', 'Cancelled')),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    paid_at TIMESTAMPTZ NULL
);

CREATE INDEX invoices_customer_id_idx ON invoices (customer_id);
