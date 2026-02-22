CREATE TABLE points_balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    program_id UUID NOT NULL REFERENCES programs(id),
    balance INTEGER NOT NULL DEFAULT 0 CHECK (balance >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(client_id, program_id)
);

CREATE TABLE points_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    program_id UUID NOT NULL REFERENCES programs(id),
    collaborator_id UUID REFERENCES collaborators(id),
    type VARCHAR(20) NOT NULL CHECK (type IN ('earn', 'burn', 'adjustment')),
    amount INTEGER NOT NULL,
    balance_after INTEGER NOT NULL,
    invoice_url TEXT,
    description TEXT,
    manual_entry BOOLEAN NOT NULL DEFAULT false,
    correction_reason TEXT,
    correction_evidence_url TEXT,
    correctable_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_client ON points_transactions(client_id);
CREATE INDEX idx_transactions_created ON points_transactions(created_at);
