CREATE TABLE IF NOT EXISTS monitoring_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL,
    user_id UUID NOT NULL,
    rule_code VARCHAR(120) NOT NULL,
    severity VARCHAR(32) NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    reason TEXT NOT NULL,
    event_time TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT monitoring_events_unique_rule_tx UNIQUE (rule_code, transaction_id)
);

CREATE INDEX IF NOT EXISTS idx_monitoring_events_user_id ON monitoring_events(user_id);
CREATE INDEX IF NOT EXISTS idx_monitoring_events_rule_code ON monitoring_events(rule_code);
CREATE INDEX IF NOT EXISTS idx_monitoring_events_created_at ON monitoring_events(created_at);
