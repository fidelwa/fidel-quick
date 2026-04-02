-- Add timezone to customers (per-business timezone)
ALTER TABLE customers ADD COLUMN IF NOT EXISTS timezone VARCHAR(50) NOT NULL DEFAULT 'America/Mexico_City';

-- Collaborator work schedules (weekly recurring)
CREATE TABLE IF NOT EXISTS collaborator_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    collaborator_id UUID NOT NULL REFERENCES collaborators(id) ON DELETE CASCADE,
    day_of_week SMALLINT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6), -- 0=Sunday
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(collaborator_id, day_of_week, start_time)
);

CREATE INDEX IF NOT EXISTS idx_collab_schedules_collab ON collaborator_schedules(collaborator_id);
