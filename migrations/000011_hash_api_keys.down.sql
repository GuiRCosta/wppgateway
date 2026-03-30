ALTER TABLE tenants ADD COLUMN api_key VARCHAR(128);

DROP INDEX IF EXISTS idx_tenants_api_key_hash;
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_api_key_hash_unique;
ALTER TABLE tenants DROP COLUMN api_key_hash;

CREATE INDEX idx_tenants_api_key ON tenants (api_key);
