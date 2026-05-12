ALTER TABLE notification_events 
DROP CONSTRAINT IF EXISTS notification_events_rule_id_fkey,
ADD CONSTRAINT notification_events_rule_id_fkey 
    FOREIGN KEY (rule_id) 
    REFERENCES notification_rules_v2(id) 
    ON DELETE CASCADE;
