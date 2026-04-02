ALTER TABLE programs ADD CONSTRAINT chk_programs_type CHECK (type IN ('earn_burn'));
ALTER TABLE cashback_programs ADD CONSTRAINT chk_cashback_programs_type CHECK (type IN ('cashback'));
