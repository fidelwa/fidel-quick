DROP INDEX IF EXISTS idx_transactions_cashback_receipt_hash;
ALTER TABLE transactions_cashback
    DROP COLUMN IF EXISTS receipt_data,
    DROP COLUMN IF EXISTS receipt_hash,
    DROP COLUMN IF EXISTS receipt_hash_fields,
    DROP COLUMN IF EXISTS receipt_confident;

DROP INDEX IF EXISTS idx_transactions_earnburn_receipt_hash;
ALTER TABLE transactions_earnburn
    DROP COLUMN IF EXISTS receipt_data,
    DROP COLUMN IF EXISTS receipt_hash,
    DROP COLUMN IF EXISTS receipt_hash_fields,
    DROP COLUMN IF EXISTS receipt_confident;
