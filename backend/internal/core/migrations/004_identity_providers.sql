ALTER TABLE users ADD COLUMN IF NOT EXISTS identity_providers jsonb NOT NULL DEFAULT '{}'::jsonb;
