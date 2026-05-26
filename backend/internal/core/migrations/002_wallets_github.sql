ALTER TABLE users ADD COLUMN IF NOT EXISTS wallet_address text NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS github_id text NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS github_username text NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS github_avatar_url text NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS users_wallet_address_idx ON users (wallet_address);
CREATE UNIQUE INDEX IF NOT EXISTS users_github_id_unique_idx ON users (github_id) WHERE github_id <> '';
CREATE UNIQUE INDEX IF NOT EXISTS users_github_username_unique_idx ON users (lower(github_username)) WHERE github_username <> '';

CREATE TABLE IF NOT EXISTS wallets (
  address text PRIMARY KEY,
  owner_user_id text NOT NULL DEFAULT '',
  github_id text NOT NULL DEFAULT '',
  github_username text NOT NULL DEFAULT '',
  recovery_salt text NOT NULL DEFAULT '',
  recovery_hash text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL,
  linked_at timestamptz
);

CREATE UNIQUE INDEX IF NOT EXISTS wallets_owner_user_id_unique_idx ON wallets (owner_user_id) WHERE owner_user_id <> '';
CREATE UNIQUE INDEX IF NOT EXISTS wallets_github_id_unique_idx ON wallets (github_id) WHERE github_id <> '';
CREATE UNIQUE INDEX IF NOT EXISTS wallets_github_username_unique_idx ON wallets (lower(github_username)) WHERE github_username <> '';
