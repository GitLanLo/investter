CREATE TABLE IF NOT EXISTS signal_outcomes (
    id BIGSERIAL PRIMARY KEY,
    signal_run_id BIGINT NOT NULL UNIQUE REFERENCES signal_runs(id) ON DELETE CASCADE,
    asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    matured_at TIMESTAMPTZ NOT NULL,
    entry_price NUMERIC(16,6) NOT NULL,
    exit_price NUMERIC(16,6) NOT NULL,
    raw_return_pct NUMERIC(12,8) NOT NULL,
    action_return_pct NUMERIC(12,8) NOT NULL,
    is_hit BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_signal_outcomes_asset_id ON signal_outcomes (asset_id);
CREATE INDEX IF NOT EXISTS idx_signal_outcomes_matured_at ON signal_outcomes (matured_at DESC);
