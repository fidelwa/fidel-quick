-- FID-35: Vida de la tarjeta (pushcard) configurable.
-- card_expiry_days define cuántos días vive una tarjeta 'open' desde su creación
-- (created_at). NULL = sin expiración (comportamiento actual por defecto).
ALTER TABLE pushcard_config
    ADD COLUMN card_expiry_days INTEGER
        CHECK (card_expiry_days IS NULL OR card_expiry_days > 0);
