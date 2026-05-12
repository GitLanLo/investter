ALTER TABLE notification_rules_v2 ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_notification_rules_v2_user_id ON notification_rules_v2(user_id);

-- Also fix notification_events to be potentially user-scoped through rule_id or signal_run_id, 
-- but rule_id already references notification_rules_v2 which is now user-scoped.
