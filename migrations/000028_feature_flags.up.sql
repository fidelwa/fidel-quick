-- FID-26: sistema de feature flags.
-- Un flag se identifica por su `key` (naming: dominio.snake_case, ej.
-- `pushcard.qr_v2`, `admin.beta_dashboard`). La resolución de si un flag está
-- activo para una petición sigue esta precedencia:
--   1. override explícito por customer  (customer_overrides[customer_id])
--   2. activación global                 (enabled_globally)
--   3. valor por defecto                 (default_value)
-- customer_overrides es un objeto JSON {"<customer_uuid>": true|false}.
CREATE TABLE feature_flags (
    key               TEXT PRIMARY KEY,
    enabled_globally  BOOLEAN NOT NULL DEFAULT false,
    customer_overrides JSONB  NOT NULL DEFAULT '{}'::jsonb,
    default_value     BOOLEAN NOT NULL DEFAULT false,
    description       TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
