CREATE TABLE platform_config (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO platform_config (key, value) VALUES
    ('whatsapp_phone_number_id', ''),
    ('platform_name', 'Fidel'),
    ('platform_url', '');
