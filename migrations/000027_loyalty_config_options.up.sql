-- FID-34 / FID-36 / FID-37: opciones de configuración de lealtad.
-- Todas las columnas son NULL por defecto => comportamiento actual intacto
-- (sin vencimiento, sin ticket mínimo, sin caps de cashback).

-- FID-34 expiración de puntos + FID-36 ticket mínimo (earn_burn)
ALTER TABLE config_earnburn
    ADD COLUMN expiry_days       INTEGER        NULL CHECK (expiry_days IS NULL OR expiry_days > 0),
    ADD COLUMN min_ticket_amount NUMERIC(12, 2) NULL CHECK (min_ticket_amount IS NULL OR min_ticket_amount >= 0);

-- FID-34 expiración de saldo + FID-36 ticket mínimo + FID-37 caps (cashback)
ALTER TABLE config_cashback
    ADD COLUMN expiry_days             INTEGER        NULL CHECK (expiry_days IS NULL OR expiry_days > 0),
    ADD COLUMN min_ticket_amount       NUMERIC(12, 2) NULL CHECK (min_ticket_amount IS NULL OR min_ticket_amount >= 0),
    ADD COLUMN max_cashback_per_tx     NUMERIC(12, 2) NULL CHECK (max_cashback_per_tx IS NULL OR max_cashback_per_tx >= 0),
    ADD COLUMN max_cashback_per_period NUMERIC(12, 2) NULL CHECK (max_cashback_per_period IS NULL OR max_cashback_per_period >= 0);
