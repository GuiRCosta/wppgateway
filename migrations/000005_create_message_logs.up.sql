CREATE TABLE message_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id        UUID REFERENCES instance_groups(id) ON DELETE SET NULL,
    instance_id     UUID REFERENCES instances(id) ON DELETE SET NULL,
    recipient       VARCHAR(20) NOT NULL,
    message_type    VARCHAR(20) NOT NULL,
    content_hash    VARCHAR(64),
    status          VARCHAR(20) NOT NULL DEFAULT 'queued'
                    CHECK (status IN ('queued', 'sent', 'delivered', 'read', 'failed')),
    error_code      VARCHAR(50),
    queued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at         TIMESTAMPTZ,
    delivered_at    TIMESTAMPTZ,
    read_at         TIMESTAMPTZ
);

CREATE INDEX idx_message_logs_group_queued ON message_logs (group_id, queued_at);
CREATE INDEX idx_message_logs_instance_status ON message_logs (instance_id, status);
CREATE INDEX idx_message_logs_status_queued ON message_logs (status) WHERE status = 'queued';
