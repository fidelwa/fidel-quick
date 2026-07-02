-- Revertir a UUID solo es seguro si todos los valores actuales son UUID-shaped.
-- Si hay descripciones de texto libre, hay que limpiarlas antes (UPDATE ... = NULL).
ALTER TABLE pushcard_config
    ALTER COLUMN reward_on_complete TYPE UUID USING reward_on_complete::uuid;
