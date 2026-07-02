DROP INDEX IF EXISTS idx_admins_hosted_domain;

ALTER TABLE admins
    DROP COLUMN IF EXISTS full_name,
    DROP COLUMN IF EXISTS avatar_url,
    DROP COLUMN IF EXISTS locale,
    DROP COLUMN IF EXISTS hosted_domain;
