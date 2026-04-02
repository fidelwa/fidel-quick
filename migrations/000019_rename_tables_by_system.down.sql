-- Revert to original table names

-- Earn-burn system
ALTER TABLE transactions_earnburn RENAME TO points_transactions;
ALTER TABLE balances_earnburn RENAME TO points_balances;
ALTER TABLE rewards_earnburn RENAME TO rewards;
ALTER TABLE redemptions_earnburn RENAME TO redemptions;
ALTER TABLE config_earnburn RENAME TO earn_burn_config;

-- Cashback system
ALTER TABLE transactions_cashback RENAME TO cashback_transactions;
ALTER TABLE balances_cashback RENAME TO cashback_balances;
ALTER TABLE rewards_cashback RENAME TO cashback_rewards;
ALTER TABLE redemptions_cashback RENAME TO cashback_redemptions;
ALTER TABLE config_cashback RENAME TO cashback_config;
