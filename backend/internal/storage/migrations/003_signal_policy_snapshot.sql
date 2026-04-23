ALTER TABLE signal_runs
    ADD COLUMN IF NOT EXISTS policy_status TEXT NULL,
    ADD COLUMN IF NOT EXISTS policy_model_name TEXT NULL,
    ADD COLUMN IF NOT EXISTS policy_scenario_name TEXT NULL,
    ADD COLUMN IF NOT EXISTS policy_calibration_method TEXT NULL,
    ADD COLUMN IF NOT EXISTS policy_threshold NUMERIC(5,4) NULL,
    ADD COLUMN IF NOT EXISTS policy_dataset_version TEXT NULL;

CREATE INDEX IF NOT EXISTS idx_signal_runs_policy_dataset_version ON signal_runs (policy_dataset_version);
CREATE INDEX IF NOT EXISTS idx_signal_runs_policy_status ON signal_runs (policy_status);
