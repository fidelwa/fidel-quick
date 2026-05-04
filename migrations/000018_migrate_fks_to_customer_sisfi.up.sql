-- Clean existing data (user confirmed data loss is acceptable)
TRUNCATE points_balances, points_transactions, rewards, redemptions CASCADE;
TRUNCATE cashback_balances, cashback_transactions, cashback_rewards, cashback_redemptions CASCADE;

-- 1. Earn-burn tables (currently FK → programs.id)
ALTER TABLE points_balances DROP CONSTRAINT points_balances_program_id_fkey;
ALTER TABLE points_balances RENAME COLUMN program_id TO customer_sisfi_id;
ALTER TABLE points_balances ADD CONSTRAINT points_balances_customer_sisfi_id_fkey
    FOREIGN KEY (customer_sisfi_id) REFERENCES customer_sisfi(id);

ALTER TABLE points_transactions DROP CONSTRAINT points_transactions_program_id_fkey;
ALTER TABLE points_transactions RENAME COLUMN program_id TO customer_sisfi_id;
ALTER TABLE points_transactions ADD CONSTRAINT points_transactions_customer_sisfi_id_fkey
    FOREIGN KEY (customer_sisfi_id) REFERENCES customer_sisfi(id);

ALTER TABLE rewards DROP CONSTRAINT rewards_program_id_fkey;
ALTER TABLE rewards RENAME COLUMN program_id TO customer_sisfi_id;
ALTER TABLE rewards ADD CONSTRAINT rewards_customer_sisfi_id_fkey
    FOREIGN KEY (customer_sisfi_id) REFERENCES customer_sisfi(id);

ALTER TABLE redemptions DROP CONSTRAINT redemptions_program_id_fkey;
ALTER TABLE redemptions RENAME COLUMN program_id TO customer_sisfi_id;
ALTER TABLE redemptions ADD CONSTRAINT redemptions_customer_sisfi_id_fkey
    FOREIGN KEY (customer_sisfi_id) REFERENCES customer_sisfi(id);

-- 2. Cashback tables (currently FK → cashback_programs.id)
ALTER TABLE cashback_balances DROP CONSTRAINT cashback_balances_program_id_fkey;
ALTER TABLE cashback_balances RENAME COLUMN program_id TO customer_sisfi_id;
ALTER TABLE cashback_balances ADD CONSTRAINT cashback_balances_customer_sisfi_id_fkey
    FOREIGN KEY (customer_sisfi_id) REFERENCES customer_sisfi(id);

ALTER TABLE cashback_transactions DROP CONSTRAINT cashback_transactions_program_id_fkey;
ALTER TABLE cashback_transactions RENAME COLUMN program_id TO customer_sisfi_id;
ALTER TABLE cashback_transactions ADD CONSTRAINT cashback_transactions_customer_sisfi_id_fkey
    FOREIGN KEY (customer_sisfi_id) REFERENCES customer_sisfi(id);

ALTER TABLE cashback_rewards DROP CONSTRAINT cashback_rewards_program_id_fkey;
ALTER TABLE cashback_rewards RENAME COLUMN program_id TO customer_sisfi_id;
ALTER TABLE cashback_rewards ADD CONSTRAINT cashback_rewards_customer_sisfi_id_fkey
    FOREIGN KEY (customer_sisfi_id) REFERENCES customer_sisfi(id);

ALTER TABLE cashback_redemptions DROP CONSTRAINT cashback_redemptions_program_id_fkey;
ALTER TABLE cashback_redemptions RENAME COLUMN program_id TO customer_sisfi_id;
ALTER TABLE cashback_redemptions ADD CONSTRAINT cashback_redemptions_customer_sisfi_id_fkey
    FOREIGN KEY (customer_sisfi_id) REFERENCES customer_sisfi(id);

-- 3. Update unique constraints
ALTER TABLE points_balances DROP CONSTRAINT IF EXISTS points_balances_client_id_program_id_key;
ALTER TABLE points_balances ADD CONSTRAINT points_balances_client_id_customer_sisfi_id_key
    UNIQUE(client_id, customer_sisfi_id);

ALTER TABLE cashback_balances DROP CONSTRAINT IF EXISTS cashback_balances_client_id_program_id_key;
ALTER TABLE cashback_balances ADD CONSTRAINT cashback_balances_client_id_customer_sisfi_id_key
    UNIQUE(client_id, customer_sisfi_id);

-- 4. Drop old tables
DROP TABLE IF EXISTS programs CASCADE;
DROP TABLE IF EXISTS cashback_programs CASCADE;
