CREATE TABLE session_credentials (
    instance_id     UUID PRIMARY KEY REFERENCES instances(id) ON DELETE CASCADE,
    creds_encrypted BYTEA NOT NULL,
    iv              BYTEA NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
