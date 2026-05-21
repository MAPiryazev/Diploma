CREATE INDEX IF NOT EXISTS idx_transactions_active_user_occurred_desc
ON transactions(user_id, occurred_at DESC)
WHERE deleted_at IS NULL;
