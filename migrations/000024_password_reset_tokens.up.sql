-- FID-16: Flujo "Olvidé mi contraseña" (reset por email).
-- Tokens de un solo uso para restablecer la password de un admin.
-- Guardamos SOLO el hash SHA-256 del token (nunca el token en claro):
-- el link enviado por email lleva el token en claro; el servidor lo
-- hashea y busca por token_hash. TTL corto (1h) y marca used_at al usarse.
CREATE TABLE password_reset_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id   UUID NOT NULL REFERENCES admins(id),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_reset_tokens_admin ON password_reset_tokens(admin_id);
CREATE INDEX idx_password_reset_tokens_expires ON password_reset_tokens(expires_at);
