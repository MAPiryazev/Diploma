ALTER TABLE accounts
ADD COLUMN IF NOT EXISTS balance NUMERIC(18, 2) NOT NULL DEFAULT 0;

WITH recalculated_balances AS (
    SELECT account_id, SUM(delta) AS balance
    FROM (
        SELECT to_account_id AS account_id, amount::numeric AS delta
        FROM transactions
        WHERE status = 'done' AND deleted_at IS NULL AND to_account_id IS NOT NULL

        UNION ALL

        SELECT from_account_id AS account_id, -amount::numeric AS delta
        FROM transactions
        WHERE status = 'done' AND deleted_at IS NULL AND from_account_id IS NOT NULL
    ) movement
    GROUP BY account_id
)
UPDATE accounts AS a
SET balance = COALESCE(rb.balance, 0)
FROM recalculated_balances AS rb
WHERE a.id = rb.account_id;

UPDATE accounts
SET balance = 0
WHERE balance IS NULL;
