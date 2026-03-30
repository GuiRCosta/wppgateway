CREATE TABLE instance_groups (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name           VARCHAR(255) NOT NULL,
    strategy       VARCHAR(20) NOT NULL DEFAULT 'failover'
                   CHECK (strategy IN ('failover', 'rotation', 'hybrid')),
    config         JSONB NOT NULL DEFAULT '{}',
    webhook_url    TEXT,
    webhook_secret VARCHAR(128),
    webhook_events TEXT[] DEFAULT '{}',
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_instance_groups_tenant_id ON instance_groups (tenant_id);
