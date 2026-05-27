ALTER TABLE gemini_api_keys
  ADD COLUMN IF NOT EXISTS provider text NOT NULL DEFAULT 'gemini',
  ADD COLUMN IF NOT EXISTS model text NOT NULL DEFAULT '';

UPDATE gemini_api_keys
SET provider = 'gemini'
WHERE provider = '';

CREATE INDEX IF NOT EXISTS gemini_api_keys_provider_status_idx ON gemini_api_keys (provider, status);
