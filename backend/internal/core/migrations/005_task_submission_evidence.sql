ALTER TABLE tasks
  ADD COLUMN IF NOT EXISTS pull_request_url text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS review_evidence_url text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS review_notes text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS submitted_at timestamptz;

CREATE INDEX IF NOT EXISTS tasks_submitted_at_idx ON tasks (submitted_at);
