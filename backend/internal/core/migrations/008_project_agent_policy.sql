ALTER TABLE projects ADD COLUMN IF NOT EXISTS allow_agents boolean NOT NULL DEFAULT true;
