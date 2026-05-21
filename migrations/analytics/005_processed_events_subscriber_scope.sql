ALTER TABLE processed_events
    ADD COLUMN IF NOT EXISTS subscriber_name VARCHAR(120);

UPDATE processed_events
SET subscriber_name = 'projection-builder'
WHERE subscriber_name IS NULL;

ALTER TABLE processed_events
    ALTER COLUMN subscriber_name SET NOT NULL;

ALTER TABLE processed_events
    DROP CONSTRAINT IF EXISTS processed_events_pkey;

ALTER TABLE processed_events
    ADD CONSTRAINT processed_events_pkey PRIMARY KEY (subscriber_name, event_id);

CREATE INDEX IF NOT EXISTS idx_processed_events_event_type ON processed_events(event_type);
CREATE INDEX IF NOT EXISTS idx_processed_events_subscriber_name ON processed_events(subscriber_name);
