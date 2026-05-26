ALTER TABLE tasks
  ADD COLUMN IF NOT EXISTS issue_state text NOT NULL DEFAULT 'open';

UPDATE tasks
SET issue_state = 'open'
WHERE issue_state IS NULL OR issue_state = '';

CREATE INDEX IF NOT EXISTS tasks_project_issue_number_idx ON tasks (project_id, issue_number);
CREATE INDEX IF NOT EXISTS tasks_issue_state_idx ON tasks (issue_state);
