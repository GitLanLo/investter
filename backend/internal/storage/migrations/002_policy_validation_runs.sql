CREATE TABLE IF NOT EXISTS policy_validation_runs (
    id BIGSERIAL PRIMARY KEY,
    policy_status TEXT NOT NULL,
    model_name TEXT NOT NULL,
    scenario_name TEXT NOT NULL,
    calibration_method TEXT NOT NULL,
    threshold NUMERIC(5,4) NOT NULL,
    dataset_version TEXT NOT NULL,
    validation_actionable_f1 NUMERIC(8,6) NOT NULL,
    validation_precision NUMERIC(8,6) NOT NULL,
    validation_coverage NUMERIC(8,6) NOT NULL,
    validation_actionable_ece NUMERIC(8,6) NOT NULL,
    test_actionable_f1 NUMERIC(8,6) NOT NULL,
    test_precision NUMERIC(8,6) NOT NULL,
    test_coverage NUMERIC(8,6) NOT NULL,
    test_actionable_ece NUMERIC(8,6) NOT NULL,
    decision_state TEXT NOT NULL DEFAULT 'candidate',
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_policy_validation_runs_created_at ON policy_validation_runs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_policy_validation_runs_decision_state ON policy_validation_runs (decision_state);
