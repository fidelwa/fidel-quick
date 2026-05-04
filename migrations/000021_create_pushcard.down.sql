DROP INDEX IF EXISTS idx_pushcard_stamps_recent;
DROP INDEX IF EXISTS idx_pushcard_stamps_card;
DROP TABLE IF EXISTS pushcard_stamps;

DROP INDEX IF EXISTS idx_pushcard_cards_one_open;
DROP INDEX IF EXISTS idx_pushcard_cards_customer_sisfi;
DROP INDEX IF EXISTS idx_pushcard_cards_client;
DROP TABLE IF EXISTS pushcard_cards;

DROP TABLE IF EXISTS pushcard_config;

DELETE FROM sisfi WHERE id = 'pushcard';
