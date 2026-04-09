CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS assets (
    id TEXT PRIMARY KEY,
    ticker TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    exchange TEXT NOT NULL DEFAULT 'MOEX',
    timeframe TEXT NOT NULL DEFAULT '5m',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS watchlists (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS watchlist_items (
    id BIGSERIAL PRIMARY KEY,
    watchlist_id BIGINT NOT NULL REFERENCES watchlists(id) ON DELETE CASCADE,
    asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (watchlist_id, asset_id)
);

CREATE TABLE IF NOT EXISTS notification_rules (
    id BIGSERIAL PRIMARY KEY,
    watchlist_id BIGINT NULL REFERENCES watchlists(id) ON DELETE CASCADE,
    asset_id TEXT NULL REFERENCES assets(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    direction TEXT NOT NULL DEFAULT 'up',
    threshold NUMERIC(5,4) NOT NULL,
    cooldown_minutes INTEGER NOT NULL DEFAULT 60,
    channel_type TEXT NOT NULL DEFAULT 'telegram',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS model_registry (
    id BIGSERIAL PRIMARY KEY,
    model_version TEXT NOT NULL UNIQUE,
    model_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'candidate',
    timeframe TEXT NOT NULL,
    horizon_bars INTEGER NOT NULL,
    feature_schema_version TEXT NOT NULL,
    manifest_path TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS signal_runs (
    id BIGSERIAL PRIMARY KEY,
    asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    model_version TEXT NOT NULL,
    as_of_time TIMESTAMPTZ NOT NULL,
    signal_state TEXT NOT NULL,
    signal_direction TEXT NOT NULL,
    signal_probability NUMERIC(8,6) NOT NULL,
    class_probabilities JSONB NOT NULL,
    threshold NUMERIC(5,4) NOT NULL,
    timeframe TEXT NOT NULL,
    horizon_bars INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS signal_events (
    id BIGSERIAL PRIMARY KEY,
    signal_run_id BIGINT NOT NULL REFERENCES signal_runs(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS job_runs (
    id BIGSERIAL PRIMARY KEY,
    job_type TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NULL
);

CREATE INDEX IF NOT EXISTS idx_watchlist_items_watchlist_id ON watchlist_items (watchlist_id);
CREATE INDEX IF NOT EXISTS idx_notification_rules_asset_id ON notification_rules (asset_id);
CREATE INDEX IF NOT EXISTS idx_signal_runs_asset_time ON signal_runs (asset_id, as_of_time DESC);
CREATE INDEX IF NOT EXISTS idx_signal_runs_model_version ON signal_runs (model_version);
CREATE INDEX IF NOT EXISTS idx_job_runs_job_type_started_at ON job_runs (job_type, started_at DESC);

INSERT INTO schema_migrations (version, name)
VALUES ('001_init', 'init')
ON CONFLICT (version) DO NOTHING;
