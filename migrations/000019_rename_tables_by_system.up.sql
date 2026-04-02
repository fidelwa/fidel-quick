-- Rename system-specific tables to {entity}_{system} convention

-- Earn-burn system
ALTER TABLE points_transactions RENAME TO transactions_earnburn;
ALTER TABLE points_balances RENAME TO balances_earnburn;
ALTER TABLE rewards RENAME TO rewards_earnburn;
ALTER TABLE redemptions RENAME TO redemptions_earnburn;
ALTER TABLE earn_burn_config RENAME TO config_earnburn;

-- Cashback system
ALTER TABLE cashback_transactions RENAME TO transactions_cashback;
ALTER TABLE cashback_balances RENAME TO balances_cashback;
ALTER TABLE cashback_rewards RENAME TO rewards_cashback;
ALTER TABLE cashback_redemptions RENAME TO redemptions_cashback;
ALTER TABLE cashback_config RENAME TO config_cashback;
