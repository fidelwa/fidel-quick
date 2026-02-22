CREATE TABLE clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    name VARCHAR(255),
    phone VARCHAR(20) NOT NULL,
    hash VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(customer_id, phone)
);

CREATE INDEX idx_clients_phone ON clients(phone);
CREATE INDEX idx_clients_customer ON clients(customer_id);
