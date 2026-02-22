CREATE TABLE cashback_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    type VARCHAR(50) NOT NULL DEFAULT 'cashback',
    name VARCHAR(255) NOT NULL,
    cashback_rate DECIMAL(5,4) NOT NULL DEFAULT 0.0500,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(customer_id, type),
    CHECK (cashback_rate > 0 AND cashback_rate <= 1)
);
