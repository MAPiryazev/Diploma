CREATE TABLE IF NOT EXISTS processed_events (
    subscriber_name VARCHAR(120) NOT NULL,
    event_id UUID NOT NULL,
    event_type VARCHAR(120) NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (subscriber_name, event_id)
);

CREATE INDEX IF NOT EXISTS idx_processed_events_event_type ON processed_events(event_type);
CREATE INDEX IF NOT EXISTS idx_processed_events_subscriber_name ON processed_events(subscriber_name);
