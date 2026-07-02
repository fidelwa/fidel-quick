-- Revierte FID-34 / FID-36 / FID-37.

ALTER TABLE config_cashback
    DROP COLUMN IF EXISTS max_cashback_per_period,
    DROP COLUMN IF EXISTS max_cashback_per_tx,
    DROP COLUMN IF EXISTS min_ticket_amount,
    DROP COLUMN IF EXISTS expiry_days;

ALTER TABLE config_earnburn
    DROP COLUMN IF EXISTS min_ticket_amount,
    DROP COLUMN IF EXISTS expiry_days;
