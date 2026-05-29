CREATE TABLE IF NOT EXISTS test_publish_settings (
  id text PRIMARY KEY,
  integration_type text NOT NULL,
  display_name text NOT NULL DEFAULT '',
  key_name text NOT NULL,
  key_value text NOT NULL,
  key_hint text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'active',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  last_used_at timestamptz
);

CREATE INDEX IF NOT EXISTS test_publish_settings_type_status_idx
  ON test_publish_settings (integration_type, status);
