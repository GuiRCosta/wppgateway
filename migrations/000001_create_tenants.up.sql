CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE tenants (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(255) NOT NULL,
    api_key       VARCHAR(128) UNIQUE NOT NULL,
    plan          VARCHAR(50) NOT NULL DEFAULT 'basic',
    max_groups    INT NOT NULL DEFAULT 5,
    max_instances INT NOT NULL DEFAULT 20,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_api_key ON tenants (api_key);
