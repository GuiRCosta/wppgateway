CREATE TABLE device_mapping (
    instance_id UUID PRIMARY KEY REFERENCES instances(id) ON DELETE CASCADE,
    jid         VARCHAR(100) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_device_mapping_jid ON device_mapping (jid);
