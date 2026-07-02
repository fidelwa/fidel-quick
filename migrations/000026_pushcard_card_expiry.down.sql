-- Revierte FID-35.
ALTER TABLE pushcard_config
    DROP COLUMN card_expiry_days;
