-- Pushcard (tarjeta de sellos) — tercer sisfi del MVP.
-- Sigue el mismo patrón modular de earn_burn y cashback:
-- una tabla de config por customer_sisfi y tablas operacionales propias.

-- Registro en el catálogo global de sistemas de fidelización.
INSERT INTO sisfi (id, name, description) VALUES
    ('pushcard', 'Tarjeta de sellos', 'Acumula sellos hasta completar la tarjeta y recibe la recompensa');

-- Configuración del sisfi pushcard por negocio.
CREATE TABLE pushcard_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_sisfi_id UUID NOT NULL UNIQUE REFERENCES customer_sisfi(id),
    card_slots INTEGER NOT NULL DEFAULT 10 CHECK (card_slots > 0),
    reward_on_complete UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tarjetas: cada tarjeta vive entre 'open' y 'completed'.
-- Una vez completada se cierra y el cliente arranca otra al sumar el siguiente sello.
CREATE TABLE pushcard_cards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_sisfi_id UUID NOT NULL REFERENCES customer_sisfi(id),
    client_id UUID NOT NULL REFERENCES clients(id),
    status VARCHAR(20) NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'completed', 'redeemed', 'cancelled')),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pushcard_cards_client ON pushcard_cards(client_id);
CREATE INDEX idx_pushcard_cards_customer_sisfi ON pushcard_cards(customer_sisfi_id);

-- Sólo una tarjeta abierta por (customer_sisfi, cliente). Lo demás puede repetirse.
CREATE UNIQUE INDEX idx_pushcard_cards_one_open
    ON pushcard_cards(customer_sisfi_id, client_id)
    WHERE status = 'open';

-- Sellos: append-only por colaborador. Un sello por fila.
CREATE TABLE pushcard_stamps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id UUID NOT NULL REFERENCES pushcard_cards(id),
    collaborator_id UUID NOT NULL REFERENCES collaborators(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pushcard_stamps_card ON pushcard_stamps(card_id);
CREATE INDEX idx_pushcard_stamps_recent
    ON pushcard_stamps(collaborator_id, created_at DESC);
