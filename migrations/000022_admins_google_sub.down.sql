ALTER TABLE admins
    DROP COLUMN IF EXISTS google_sub,
    DROP COLUMN IF EXISTS google_email;
