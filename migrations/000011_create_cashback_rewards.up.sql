CREATE TABLE cashback_rewards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    program_id UUID NOT NULL REFERENCES cashback_programs(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    cost DECIMAL(12,2) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cashback_rewards_customer ON cashback_rewards(customer_id);

CREATE TABLE cashback_redemptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id),
    reward_id UUID NOT NULL REFERENCES cashback_rewards(id),
    program_id UUID NOT NULL REFERENCES cashback_programs(id),
    code VARCHAR(20) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'confirmed', 'expired', 'cancelled'
    amount_spent DECIMAL(12,2) NOT NULL,
    confirmed_by UUID REFERENCES collaborators(id),
    expires_at TIMESTAMPTZ NOT NULL,
    confirmed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cashback_redemptions_code ON cashback_redemptions(code);
CREATE INDEX idx_cashback_redemptions_client ON cashback_redemptions(client_id);
CREATE INDEX idx_cashback_redemptions_pending ON cashback_redemptions(status) WHERE status = 'pending';
