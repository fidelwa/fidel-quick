-- Anti-fraude (FID-33): persistir los datos completos del ticket extraídos por IA
-- y deduplicar por hash canónico. Se guarda el InvoiceResult íntegro en JSONB y un
-- hash SHA-256 de un subconjunto canónico normalizado para detectar tickets repetidos.
--
-- Decisión (doc de arquitectura): NO se crea una base de datos nueva; se extienden
-- las tablas operacionales existentes con columnas JSONB + hash.

-- earn_burn -----------------------------------------------------------------
ALTER TABLE transactions_earnburn
    ADD COLUMN receipt_data        JSONB,
    ADD COLUMN receipt_hash        TEXT,
    ADD COLUMN receipt_hash_fields TEXT[],
    ADD COLUMN receipt_confident   BOOLEAN;

-- Un mismo ticket (hash) no puede acreditarse dos veces dentro del mismo
-- customer_sisfi (negocio + programa). El índice es parcial: cuando no hay hash
-- confiable receipt_hash queda NULL y no participa en la unicidad.
CREATE UNIQUE INDEX idx_transactions_earnburn_receipt_hash
    ON transactions_earnburn (customer_sisfi_id, receipt_hash)
    WHERE receipt_hash IS NOT NULL;

-- cashback ------------------------------------------------------------------
ALTER TABLE transactions_cashback
    ADD COLUMN receipt_data        JSONB,
    ADD COLUMN receipt_hash        TEXT,
    ADD COLUMN receipt_hash_fields TEXT[],
    ADD COLUMN receipt_confident   BOOLEAN;

CREATE UNIQUE INDEX idx_transactions_cashback_receipt_hash
    ON transactions_cashback (customer_sisfi_id, receipt_hash)
    WHERE receipt_hash IS NOT NULL;
