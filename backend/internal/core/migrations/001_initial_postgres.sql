CREATE TABLE IF NOT EXISTS schema_migrations (
  version text PRIMARY KEY,
  applied_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS store_meta (
  key text PRIMARY KEY,
  value text NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
  id text PRIMARY KEY,
  name text NOT NULL,
  company_name text NOT NULL DEFAULT '',
  email text NOT NULL UNIQUE,
  role text NOT NULL,
  password_salt text NOT NULL,
  password_hash text NOT NULL,
  created_at timestamptz NOT NULL,
  last_login_at timestamptz
);

CREATE TABLE IF NOT EXISTS projects (
  id text PRIMARY KEY,
  client_user_id text NOT NULL,
  title text NOT NULL,
  client_name text NOT NULL,
  company_name text NOT NULL DEFAULT '',
  client_email text NOT NULL,
  phone text NOT NULL DEFAULT '',
  site_type text NOT NULL DEFAULT '',
  package_tier text NOT NULL DEFAULT '',
  timeline text NOT NULL DEFAULT '',
  brief text NOT NULL DEFAULT '',
  payment_method text NOT NULL,
  payment_status text NOT NULL,
  payment_provider text NOT NULL DEFAULT '',
  payment_reference text NOT NULL DEFAULT '',
  bounty_repo_name text NOT NULL DEFAULT '',
  repo_visibility text NOT NULL DEFAULT '',
  repo_provider text NOT NULL DEFAULT '',
  repo_url text NOT NULL DEFAULT '',
  repo_local_path text NOT NULL DEFAULT '',
  budget_cents bigint NOT NULL,
  fee_cents bigint NOT NULL,
  work_pool_cents bigint NOT NULL,
  status text NOT NULL,
  created_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS projects_client_user_id_idx ON projects (client_user_id);
CREATE INDEX IF NOT EXISTS projects_created_at_idx ON projects (created_at DESC);

CREATE TABLE IF NOT EXISTS tasks (
  id text PRIMARY KEY,
  project_id text NOT NULL,
  issue_number integer NOT NULL,
  title text NOT NULL,
  acceptance text NOT NULL,
  reward_cents bigint NOT NULL,
  required_worker_kind text NOT NULL,
  suggested_agent_type text NOT NULL DEFAULT '',
  status text NOT NULL,
  worker_kind text NOT NULL DEFAULT '',
  worker_id text NOT NULL DEFAULT '',
  agent_type text NOT NULL DEFAULT '',
  proof_hash text NOT NULL DEFAULT '',
  issue_url text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL,
  accepted_at timestamptz
);

CREATE INDEX IF NOT EXISTS tasks_project_id_idx ON tasks (project_id);
CREATE INDEX IF NOT EXISTS tasks_status_idx ON tasks (status);

CREATE TABLE IF NOT EXISTS sessions (
  token text PRIMARY KEY,
  user_id text NOT NULL,
  created_at timestamptz NOT NULL,
  expires_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_user_id_idx ON sessions (user_id);
CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE IF NOT EXISTS notifications (
  id text PRIMARY KEY,
  user_id text NOT NULL,
  project_id text NOT NULL DEFAULT '',
  channel text NOT NULL,
  subject text NOT NULL,
  body text NOT NULL,
  status text NOT NULL,
  created_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS notifications_user_id_idx ON notifications (user_id);
CREATE INDEX IF NOT EXISTS notifications_created_at_idx ON notifications (created_at DESC);

CREATE TABLE IF NOT EXISTS attachments (
  id text PRIMARY KEY,
  user_id text NOT NULL DEFAULT '',
  project_id text NOT NULL DEFAULT '',
  original_name text NOT NULL,
  stored_name text NOT NULL,
  content_type text NOT NULL,
  size_bytes bigint NOT NULL,
  url text NOT NULL,
  stored_path text NOT NULL DEFAULT '',
  is_image boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS attachments_user_id_idx ON attachments (user_id);
CREATE INDEX IF NOT EXISTS attachments_project_id_idx ON attachments (project_id);

CREATE TABLE IF NOT EXISTS ssl_reviews (
  domain text PRIMARY KEY,
  port text NOT NULL,
  status text NOT NULL,
  issuer text NOT NULL DEFAULT '',
  subject text NOT NULL DEFAULT '',
  serial_number text NOT NULL DEFAULT '',
  dns_names jsonb NOT NULL DEFAULT '[]'::jsonb,
  not_before timestamptz,
  not_after timestamptz,
  days_remaining integer NOT NULL DEFAULT 0,
  last_checked_at timestamptz,
  next_check_at timestamptz,
  error text NOT NULL DEFAULT '',
  checked_by text NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS ledger_entries (
  sequence integer PRIMARY KEY,
  type text NOT NULL,
  from_account text NOT NULL DEFAULT '',
  to_account text NOT NULL DEFAULT '',
  amount_cents bigint NOT NULL,
  reference text NOT NULL,
  previous_hash text NOT NULL,
  entry_hash text NOT NULL,
  created_at timestamptz NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS ledger_entries_entry_hash_idx ON ledger_entries (entry_hash);
CREATE INDEX IF NOT EXISTS ledger_entries_created_at_idx ON ledger_entries (created_at);
