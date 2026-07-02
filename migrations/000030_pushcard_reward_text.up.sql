-- FID-14: pushcard.reward_on_complete pasa de UUID a TEXT.
-- El campo siempre se almacenó como UUID pero el flujo de redención usa un
-- string libre (RewardName) y la columna nunca se consultó como FK a rewards
-- (los rewards requieren un program_id de earn-burn, que la pushcard no tiene).
-- Se vuelve TEXT para que el admin describa la recompensa libremente desde el
-- wizard de onboarding (ej. "Café gratis", "1 corte gratis") sin salir del flujo.
ALTER TABLE pushcard_config
    ALTER COLUMN reward_on_complete TYPE TEXT USING reward_on_complete::text;
