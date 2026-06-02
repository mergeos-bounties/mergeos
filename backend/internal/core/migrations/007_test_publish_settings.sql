CREATE TABLE IF NOT EXISTS test_settings_config (
  id text PRIMARY KEY,
  test_mode_enabled boolean NOT NULL DEFAULT false,
  test_password_hash text NOT NULL DEFAULT '',
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS test_settings_entries (
  id text PRIMARY KEY,
  integration_type text NOT NULL,
  display_name text NOT NULL DEFAULT '',
  setting_key text NOT NULL,
  setting_value text NOT NULL,
  key_value_map jsonb NOT NULL DEFAULT '{}',
  status text NOT NULL DEFAULT 'active',
  last_used_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS test_settings_entries_type_idx ON test_settings_entries (integration_type);
CREATE INDEX IF NOT EXISTS test_settings_entries_status_idx ON test_settings_entries (status);
