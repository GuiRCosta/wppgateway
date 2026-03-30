CREATE TABLE blacklist (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id    UUID NOT NULL REFERENCES instance_groups(id) ON DELETE CASCADE,
    phone       VARCHAR(20) NOT NULL,
    reason      VARCHAR(50) DEFAULT 'opt_out',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (group_id, phone)
);

CREATE INDEX idx_blacklist_group_phone ON blacklist (group_id, phone);
