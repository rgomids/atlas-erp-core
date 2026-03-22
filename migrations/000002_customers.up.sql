CREATE TABLE customers (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    document VARCHAR(14) NOT NULL,
    email TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('Active', 'Inactive')),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX customers_document_uq ON customers (document);
