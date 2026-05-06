-- Revertir solo es seguro si todos los valores actuales son UUID-shaped.
-- Si hay textos libres, este down falla — usar TRUNCATE/UPDATE primero.
ALTER TABLE pushcard_config
    ALTER COLUMN reward_on_complete TYPE UUID USING reward_on_complete::uuid;
