-- This is a destructive migration - full rollback requires recreating programs/cashback_programs
-- For safety, this down migration only renames columns back

ALTER TABLE points_balances RENAME COLUMN customer_sisfi_id TO program_id;
ALTER TABLE points_transactions RENAME COLUMN customer_sisfi_id TO program_id;
ALTER TABLE rewards RENAME COLUMN customer_sisfi_id TO program_id;
ALTER TABLE redemptions RENAME COLUMN customer_sisfi_id TO program_id;
ALTER TABLE cashback_balances RENAME COLUMN customer_sisfi_id TO program_id;
ALTER TABLE cashback_transactions RENAME COLUMN customer_sisfi_id TO program_id;
ALTER TABLE cashback_rewards RENAME COLUMN customer_sisfi_id TO program_id;
ALTER TABLE cashback_redemptions RENAME COLUMN customer_sisfi_id TO program_id;
