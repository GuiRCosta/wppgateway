CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE tenants ADD COLUMN api_key_hash VARCHAR(64);

UPDATE tenants SET api_key_hash = encode(digest(api_key, 'sha256'), 'hex') WHERE api_key IS NOT NULL;

ALTER TABLE tenants ALTER COLUMN api_key_hash SET NOT NULL;
ALTER TABLE tenants ADD CONSTRAINT tenants_api_key_hash_unique UNIQUE (api_key_hash);

DROP INDEX IF EXISTS idx_tenants_api_key;
ALTER TABLE tenants DROP COLUMN api_key;

CREATE INDEX idx_tenants_api_key_hash ON tenants (api_key_hash);
