-- Идемпотентные POST: один ключ на пару (user_id, idempotency_key), тело фиксируется хешем
CREATE TABLE IF NOT EXISTS idempotency_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    idempotency_key TEXT NOT NULL,
    body_hash BYTEA NOT NULL,
    http_status INT NOT NULL,
    response_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT idempotency_keys_key_len CHECK (char_length(idempotency_key) <= 255),
    CONSTRAINT idempotency_keys_unique_user_key UNIQUE (user_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_idempotency_keys_user_id ON idempotency_keys(user_id);
