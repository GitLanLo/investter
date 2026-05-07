CREATE TABLE IF NOT EXISTS policy_promotion_log (
    id BIGSERIAL PRIMARY KEY,
    policy_validation_run_id BIGINT NOT NULL REFERENCES policy_validation_runs(id),
    actor TEXT NOT NULL DEFAULT 'system',
    previous_state TEXT NOT NULL,
    next_state TEXT NOT NULL,
    blockers JSONB,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_policy_promotion_log_run_id ON policy_promotion_log (policy_validation_run_id);
CREATE INDEX IF NOT EXISTS idx_policy_promotion_log_created_at ON policy_promotion_log (created_at DESC);
