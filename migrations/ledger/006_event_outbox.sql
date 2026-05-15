CREATE TABLE IF NOT EXISTS event_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL UNIQUE,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(120) NOT NULL,
    topic VARCHAR(255) NOT NULL,
    message_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'sent', 'failed')),
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    locked_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE event_outbox ADD COLUMN IF NOT EXISTS aggregate_id UUID;
UPDATE event_outbox SET aggregate_id = event_id WHERE aggregate_id IS NULL;
ALTER TABLE event_outbox ALTER COLUMN aggregate_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_event_outbox_ready
ON event_outbox(status, next_attempt_at, created_at)
WHERE status IN ('pending', 'failed');

CREATE INDEX IF NOT EXISTS idx_event_outbox_stale_processing
ON event_outbox(locked_at)
WHERE status = 'processing';

CREATE INDEX IF NOT EXISTS idx_event_outbox_event_type ON event_outbox(event_type);
CREATE INDEX IF NOT EXISTS idx_event_outbox_aggregate_id ON event_outbox(aggregate_id);
