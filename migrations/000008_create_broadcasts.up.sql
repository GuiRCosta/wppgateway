CREATE TABLE broadcasts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id        UUID NOT NULL REFERENCES instance_groups(id) ON DELETE CASCADE,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'processing', 'paused', 'completed', 'cancelled', 'failed')),
    total           INT NOT NULL DEFAULT 0,
    sent            INT NOT NULL DEFAULT 0,
    delivered       INT NOT NULL DEFAULT 0,
    read            INT NOT NULL DEFAULT 0,
    failed          INT NOT NULL DEFAULT 0,
    message_type    VARCHAR(20) NOT NULL,
    content         JSONB NOT NULL,
    variables       JSONB DEFAULT '{}',
    options         JSONB DEFAULT '{}',
    schedule_at     TIMESTAMPTZ,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_broadcasts_group_id ON broadcasts (group_id);
CREATE INDEX idx_broadcasts_status ON broadcasts (status);

CREATE TABLE broadcast_recipients (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    broadcast_id    UUID NOT NULL REFERENCES broadcasts(id) ON DELETE CASCADE,
    recipient       VARCHAR(20) NOT NULL,
    instance_id     UUID REFERENCES instances(id) ON DELETE SET NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'sent', 'delivered', 'read', 'failed', 'skipped')),
    error_code      VARCHAR(50),
    sent_at         TIMESTAMPTZ,
    delivered_at    TIMESTAMPTZ
);

CREATE INDEX idx_broadcast_recipients_broadcast ON broadcast_recipients (broadcast_id, status);
