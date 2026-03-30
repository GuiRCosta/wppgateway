CREATE TYPE instance_status AS ENUM (
    'disconnected',
    'connecting',
    'available',
    'resting',
    'warming',
    'suspect',
    'banned'
);

CREATE TABLE instances (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id        UUID NOT NULL REFERENCES instance_groups(id) ON DELETE CASCADE,
    phone_number    VARCHAR(20),
    display_name    VARCHAR(255),
    status          instance_status NOT NULL DEFAULT 'disconnected',
    priority        INT NOT NULL DEFAULT 0,
    daily_budget    INT NOT NULL DEFAULT 200,
    hourly_budget   INT NOT NULL DEFAULT 30,
    warmup_days     INT NOT NULL DEFAULT 0,
    messages_today  INT NOT NULL DEFAULT 0,
    messages_hour   INT NOT NULL DEFAULT 0,
    delivery_rate   FLOAT NOT NULL DEFAULT 1.0,
    last_active_at  TIMESTAMPTZ,
    banned_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_instances_group_id ON instances (group_id);
CREATE INDEX idx_instances_status ON instances (status);
