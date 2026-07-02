-- FID-38: revertir stock / disponibilidad limitada de rewards.

ALTER TABLE rewards_cashback
    DROP CONSTRAINT IF EXISTS rewards_cashback_redeemed_within_stock,
    DROP CONSTRAINT IF EXISTS rewards_cashback_stock_nonneg;

ALTER TABLE rewards_earnburn
    DROP CONSTRAINT IF EXISTS rewards_earnburn_redeemed_within_stock,
    DROP CONSTRAINT IF EXISTS rewards_earnburn_stock_nonneg;

ALTER TABLE rewards_cashback
    DROP COLUMN IF EXISTS limit_per_client,
    DROP COLUMN IF EXISTS redeemed_count,
    DROP COLUMN IF EXISTS stock_total;

ALTER TABLE rewards_earnburn
    DROP COLUMN IF EXISTS limit_per_client,
    DROP COLUMN IF EXISTS redeemed_count,
    DROP COLUMN IF EXISTS stock_total;
