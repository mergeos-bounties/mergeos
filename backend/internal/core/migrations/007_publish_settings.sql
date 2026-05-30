CREATE TABLE IF NOT EXISTS test_integration_keys (
  id text PRIMARY KEY,
  "group" text NOT NULL,
  display_name text NOT NULL,
  key_values jsonb NOT NULL DEFAULT '[]',
  status text NOT NULL DEFAULT 'active',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  last_used_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_test_integration_keys_group ON test_integration_keys("group");
