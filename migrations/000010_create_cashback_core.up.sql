CREATE TABLE cashback_balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    program_id UUID NOT NULL REFERENCES cashback_programs(id),
    balance DECIMAL(12,2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(client_id, program_id),
    CHECK (balance >= 0)
);

CREATE TABLE cashback_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    program_id UUID NOT NULL REFERENCES cashback_programs(id),
    collaborator_id UUID REFERENCES collaborators(id),
    type VARCHAR(20) NOT NULL, -- 'earn', 'burn', 'adjustment'
    amount DECIMAL(12,2) NOT NULL,
    purchase_amount DECIMAL(12,2),
    balance_after DECIMAL(12,2) NOT NULL,
    invoice_url TEXT,
    description TEXT,
    manual_entry BOOLEAN NOT NULL DEFAULT false,
    correction_reason TEXT,
    correction_evidence_url TEXT,
    correctable_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cashback_transactions_client ON cashback_transactions(client_id);
CREATE INDEX idx_cashback_transactions_created ON cashback_transactions(created_at);
