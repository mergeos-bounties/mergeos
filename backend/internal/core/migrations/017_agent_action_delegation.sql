ALTER TABLE gemini_webhook_logs
  ADD COLUMN IF NOT EXISTS delegated_by text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS design_agent text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS subagent_type text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS delegation_chain jsonb NOT NULL DEFAULT '[]'::jsonb;
