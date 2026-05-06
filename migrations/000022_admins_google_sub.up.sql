-- FID-11: vincular cuenta Google al admin (account linking + login).
-- google_sub es el identificador estable del usuario Google (claim "sub" del ID token).
-- google_email se guarda para mostrar en el panel cuál cuenta está vinculada.
ALTER TABLE admins
    ADD COLUMN google_sub TEXT UNIQUE,
    ADD COLUMN google_email TEXT;
