ALTER TABLE policy_validation_runs ADD COLUMN IF NOT EXISTS model_version TEXT NOT NULL DEFAULT '';
