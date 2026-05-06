-- FID-14: pushcard.reward_on_complete pasa de UUID a TEXT.
-- El campo siempre fue almacenado como UUID pero el flujo de redención usa
-- un string libre (RewardName) y el column nunca se consultó como FK. Se
-- vuelve TEXT para que el admin describa la recompensa libremente desde
-- el wizard de onboarding (ej. "Cafe gratis", "1 corte gratis").
ALTER TABLE pushcard_config
    ALTER COLUMN reward_on_complete TYPE TEXT USING reward_on_complete::text;
