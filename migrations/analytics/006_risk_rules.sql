CREATE TABLE IF NOT EXISTS risk_rules (
    rule_code VARCHAR(120) PRIMARY KEY,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    severity VARCHAR(32) NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    params JSONB NOT NULL DEFAULT '{}'::jsonb,
    version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO risk_rules (rule_code, enabled, severity, params, version)
VALUES
    (
        'large_amount',
        TRUE,
        'warning',
        '{"threshold_amount": 100000}'::jsonb,
        1
    ),
    (
        'velocity_1h',
        TRUE,
        'warning',
        '{"window_minutes": 60, "max_transactions": 5}'::jsonb,
        1
    ),
    (
        'velocity_24h_amount',
        TRUE,
        'critical',
        '{"window_minutes": 1440, "max_total_amount": 250000}'::jsonb,
        1
    ),
    (
        'night_activity',
        TRUE,
        'info',
        '{"night_start_hour": 0, "night_end_hour": 6, "min_amount": 10000}'::jsonb,
        1
    ),
    (
        'round_amount',
        TRUE,
        'info',
        '{"round_modulo": 1000, "min_amount": 10000}'::jsonb,
        1
    ),
    (
        'repeated_amount_24h',
        TRUE,
        'warning',
        '{"window_minutes": 1440, "repeated_transactions": 3}'::jsonb,
        1
    )
ON CONFLICT (rule_code) DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_monitoring_events_event_time
ON monitoring_events(event_time DESC);

CREATE INDEX IF NOT EXISTS idx_monitoring_events_user_event_time
ON monitoring_events(user_id, event_time DESC);

CREATE INDEX IF NOT EXISTS idx_monitoring_events_severity_event_time
ON monitoring_events(severity, event_time DESC);
