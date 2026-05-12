ALTER TABLE signal_events ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_signal_events_user_id ON signal_events(user_id);
