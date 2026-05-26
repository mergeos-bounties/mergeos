CREATE TABLE IF NOT EXISTS gemini_api_keys (
  id text PRIMARY KEY,
  key_value text NOT NULL,
  key_hint text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'active',
  request_count bigint NOT NULL DEFAULT 0,
  success_count bigint NOT NULL DEFAULT 0,
  quota_error_count bigint NOT NULL DEFAULT 0,
  last_status_code integer NOT NULL DEFAULT 0,
  last_error text NOT NULL DEFAULT '',
  last_used_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS gemini_api_keys_status_idx ON gemini_api_keys (status);
CREATE INDEX IF NOT EXISTS gemini_api_keys_request_count_idx ON gemini_api_keys (request_count, last_used_at);
