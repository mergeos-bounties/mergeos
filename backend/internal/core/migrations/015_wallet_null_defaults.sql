UPDATE wallets
SET chain = 'solana'
WHERE chain IS NULL OR btrim(chain) = '';

UPDATE wallets
SET legacy_address = ''
WHERE legacy_address IS NULL;

ALTER TABLE wallets
  ALTER COLUMN chain SET DEFAULT 'solana',
  ALTER COLUMN chain SET NOT NULL,
  ALTER COLUMN legacy_address SET DEFAULT '',
  ALTER COLUMN legacy_address SET NOT NULL;
