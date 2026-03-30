CREATE TABLE instance_metrics (
    instance_id        UUID NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    date               DATE NOT NULL,
    messages_sent      INT NOT NULL DEFAULT 0,
    messages_delivered INT NOT NULL DEFAULT 0,
    messages_failed    INT NOT NULL DEFAULT 0,
    delivery_rate      FLOAT NOT NULL DEFAULT 0,
    avg_delivery_time_ms INT NOT NULL DEFAULT 0,
    PRIMARY KEY (instance_id, date)
);
