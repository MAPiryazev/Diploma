CREATE TABLE IF NOT EXISTS transaction_event_stats (
    stat_date DATE NOT NULL,
    user_id UUID NOT NULL,
    currency CHAR(3) NOT NULL,
    created_count BIGINT NOT NULL DEFAULT 0,
    updated_count BIGINT NOT NULL DEFAULT 0,
    deleted_count BIGINT NOT NULL DEFAULT 0,
    status_changed_count BIGINT NOT NULL DEFAULT 0,
    created_amount NUMERIC(18, 2) NOT NULL DEFAULT 0,
    last_event_time TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (stat_date, user_id, currency)
);

CREATE INDEX IF NOT EXISTS idx_transaction_event_stats_user_date
ON transaction_event_stats(user_id, stat_date);
