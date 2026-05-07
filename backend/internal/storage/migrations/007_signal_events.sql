ALTER TABLE signal_events 
ADD COLUMN IF NOT EXISTS model_version TEXT NOT NULL DEFAULT '',
ADD COLUMN IF NOT EXISTS ticker TEXT,
ADD COLUMN IF NOT EXISTS idempotency_key TEXT;

ALTER TABLE signal_events ALTER COLUMN signal_run_id DROP NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_signal_events_idempotency_key ON signal_events (idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_signal_events_created_at_new ON signal_events (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_signal_events_event_type_new ON signal_events (event_type);
CREATE INDEX IF NOT EXISTS idx_signal_events_model_version ON signal_events (model_version);
