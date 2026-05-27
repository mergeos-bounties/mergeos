CREATE TABLE IF NOT EXISTS usdt_webhook_events (
    id text PRIMARY KEY,
    provider text NOT NULL DEFAULT '',
    event_type text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'pending',
    gateway_id text NOT NULL DEFAULT '',
    amount_cents bigint NOT NULL DEFAULT 0,
    currency text NOT NULL DEFAULT 'USD',
    network text NOT NULL DEFAULT '',
    tx_hash text NOT NULL DEFAULT '',
    sender_address text NOT NULL DEFAULT '',
    receiver_address text NOT NULL DEFAULT '',
    signature_valid boolean NOT NULL DEFAULT false,
    raw_payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    error text NOT NULL DEFAULT '',
    project_id text NOT NULL DEFAULT '',
    idempotency_key text NOT NULL DEFAULT '',
    processed_at timestamptz,
    received_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS usdt_webhook_events_idempotency_idx ON usdt_webhook_events (idempotency_key) WHERE idempotency_key <> '';
CREATE INDEX IF NOT EXISTS usdt_webhook_events_status_idx ON usdt_webhook_events (status);
CREATE INDEX IF NOT EXISTS usdt_webhook_events_gateway_id_idx ON usdt_webhook_events (gateway_id);
CREATE INDEX IF NOT EXISTS usdt_webhook_events_received_at_idx ON usdt_webhook_events (received_at DESC);
CREATE INDEX IF NOT EXISTS usdt_webhook_events_project_id_idx ON usdt_webhook_events (project_id);
