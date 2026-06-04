ALTER TABLE wallets
  ADD COLUMN IF NOT EXISTS chain text NOT NULL DEFAULT 'solana',
  ADD COLUMN IF NOT EXISTS legacy_address text;
