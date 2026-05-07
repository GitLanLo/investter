CREATE TABLE IF NOT EXISTS notification_rules_v2 (
    id BIGSERIAL PRIMARY KEY,
    ticker TEXT,
    event_type TEXT NOT NULL,
    severity TEXT NOT NULL,
    direction TEXT,
    model_version TEXT,
    threshold NUMERIC(8,6),
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    cooldown_minutes INT NOT NULL DEFAULT 60,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS notification_events (
    id BIGSERIAL PRIMARY KEY,
    rule_id BIGINT REFERENCES notification_rules_v2(id),
    signal_event_id BIGINT REFERENCES signal_events(id),
    event_type TEXT NOT NULL,
    severity TEXT NOT NULL,
    model_version TEXT NOT NULL,
    ticker TEXT,
    message TEXT NOT NULL,
    payload JSONB,
    delivery_status TEXT NOT NULL DEFAULT 'pending',
    delivery_attempts INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (rule_id, signal_event_id)
);

CREATE INDEX IF NOT EXISTS idx_notification_events_created_at ON notification_events (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notification_rules_v2_event_type ON notification_rules_v2 (event_type);
