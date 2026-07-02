-- FID-15: capturar perfil Google del admin (nombre, avatar, locale, hosted_domain).
-- Estos campos vienen en el ID token de Google con scopes `openid email profile`
-- y se almacenan para personalizar el panel y detectar tenants Workspace.
ALTER TABLE admins
    ADD COLUMN full_name TEXT,
    ADD COLUMN avatar_url TEXT,
    ADD COLUMN locale TEXT,
    ADD COLUMN hosted_domain TEXT;

CREATE INDEX IF NOT EXISTS idx_admins_hosted_domain ON admins(hosted_domain) WHERE hosted_domain IS NOT NULL;
