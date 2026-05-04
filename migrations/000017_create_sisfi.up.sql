-- Catálogo de sistemas de fidelización
CREATE TABLE sisfi (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO sisfi (id, name, description) VALUES
('earn_burn', 'Puntos', 'Acumula puntos por compras y canjealos por recompensas'),
('cashback', 'Cashback', 'Recibe un porcentaje de vuelta por cada compra');

-- Vinculación: qué negocios tienen qué sistemas activos
CREATE TABLE customer_sisfi (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id),
    sisfi_id VARCHAR(50) NOT NULL REFERENCES sisfi(id),
    name VARCHAR(255) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(customer_id, sisfi_id)
);

CREATE INDEX idx_customer_sisfi_customer ON customer_sisfi(customer_id);

-- Configuración earn_burn (dominio propio)
CREATE TABLE earn_burn_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_sisfi_id UUID NOT NULL UNIQUE REFERENCES customer_sisfi(id),
    points_ratio INTEGER NOT NULL DEFAULT 1000 CHECK (points_ratio > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Configuración cashback (dominio propio)
CREATE TABLE cashback_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_sisfi_id UUID NOT NULL UNIQUE REFERENCES customer_sisfi(id),
    cashback_rate DECIMAL(5,4) NOT NULL DEFAULT 0.0500 CHECK (cashback_rate > 0 AND cashback_rate <= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
