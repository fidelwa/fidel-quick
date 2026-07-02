-- FID-38: Stock / disponibilidad limitada de rewards (earn_burn / cashback).
--
-- stock_total     NULL  = stock ilimitado (comportamiento por defecto, sin cambio).
--                 N     = solo se pueden canjear N unidades del premio.
-- redeemed_count        = contador de unidades ya canjeadas; se incrementa de forma
--                         ATÓMICA dentro de la misma transacción del burn
--                         (UPDATE ... WHERE redeemed_count < stock_total), lo que
--                         garantiza que dos clientes que pelean por la última unidad
--                         no puedan ganar ambos (solo una fila afectada gana).
-- limit_per_client NULL = sin límite por cliente (opcional; reservado para uso futuro).

ALTER TABLE rewards_earnburn
    ADD COLUMN stock_total      INTEGER NULL,
    ADD COLUMN redeemed_count   INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN limit_per_client INTEGER NULL;

ALTER TABLE rewards_cashback
    ADD COLUMN stock_total      INTEGER NULL,
    ADD COLUMN redeemed_count   INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN limit_per_client INTEGER NULL;

-- Defensa en profundidad: nunca permitir sobrecanje ni valores negativos.
ALTER TABLE rewards_earnburn
    ADD CONSTRAINT rewards_earnburn_stock_nonneg CHECK (stock_total IS NULL OR stock_total >= 0),
    ADD CONSTRAINT rewards_earnburn_redeemed_within_stock CHECK (stock_total IS NULL OR redeemed_count <= stock_total);

ALTER TABLE rewards_cashback
    ADD CONSTRAINT rewards_cashback_stock_nonneg CHECK (stock_total IS NULL OR stock_total >= 0),
    ADD CONSTRAINT rewards_cashback_redeemed_within_stock CHECK (stock_total IS NULL OR redeemed_count <= stock_total);
