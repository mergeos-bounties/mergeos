CREATE TABLE IF NOT EXISTS payment_order_intents (
  order_id text PRIMARY KEY,
  provider text NOT NULL,
  flow text NOT NULL,
  user_id text NOT NULL,
  project_id text NOT NULL DEFAULT '',
  suggested_task_id text NOT NULL DEFAULT '',
  amount_cents bigint NOT NULL,
  currency text NOT NULL DEFAULT 'USD',
  description text NOT NULL DEFAULT '',
  status text NOT NULL,
  approval_url text NOT NULL DEFAULT '',
  return_url text NOT NULL DEFAULT '',
  cancel_url text NOT NULL DEFAULT '',
  capture_id text NOT NULL DEFAULT '',
  webhook_event_id text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  captured_at timestamptz
);

CREATE INDEX IF NOT EXISTS payment_order_intents_user_id_idx ON payment_order_intents (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS payment_order_intents_project_id_idx ON payment_order_intents (project_id) WHERE project_id <> '';
CREATE INDEX IF NOT EXISTS payment_order_intents_webhook_event_id_idx ON payment_order_intents (webhook_event_id) WHERE webhook_event_id <> '';
