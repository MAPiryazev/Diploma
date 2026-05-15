CREATE TABLE IF NOT EXISTS analytics_transactions (
    transaction_id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    amount NUMERIC(18, 2) NOT NULL CHECK (amount > 0),
    currency CHAR(3) NOT NULL,
    from_account_id UUID,
    to_account_id UUID,
    provider_id UUID,
    category_id UUID,
    type VARCHAR(20) NOT NULL CHECK (type IN ('income', 'expense', 'transfer')),
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'done', 'failed')),
    description TEXT,
    external_id VARCHAR(255),
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_analytics_transactions_user_occurred_at
ON analytics_transactions(user_id, occurred_at);

CREATE INDEX IF NOT EXISTS idx_analytics_transactions_user_occurred_active
ON analytics_transactions(user_id, occurred_at DESC)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_analytics_transactions_status
ON analytics_transactions(status);

CREATE INDEX IF NOT EXISTS idx_analytics_transactions_currency
ON analytics_transactions(currency);

CREATE INDEX IF NOT EXISTS idx_analytics_transactions_deleted_at
ON analytics_transactions(deleted_at);
