-- Normalize existing data
UPDATE programs SET type = 'earn_burn' WHERE type = 'earn-burn';

-- Add constraint to prevent inconsistent values
ALTER TABLE programs ADD CONSTRAINT chk_programs_type CHECK (type IN ('earn_burn'));
ALTER TABLE cashback_programs ADD CONSTRAINT chk_cashback_programs_type CHECK (type IN ('cashback'));
