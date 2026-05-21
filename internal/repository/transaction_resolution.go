package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
)

func (r *transactionRepository) ResolvePending(
	ctx context.Context,
	id string,
	approved bool,
) (*models.Transaction, bool, error) {
	dbTx, err := r.db.Master.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, false, fmt.Errorf("begin resolve transaction tx: %w", err)
	}
	defer func() { _ = dbTx.Rollback() }()

	before, err := r.getByIDForResolution(ctx, dbTx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if before.DeletedAt != nil || !isPendingStatus(before.Status) {
		if err := dbTx.Commit(); err != nil {
			return nil, false, fmt.Errorf("commit no-op resolve transaction tx: %w", err)
		}
		return before, false, nil
	}

	targetStatus := "failed"
	if approved {
		executed, err := r.tryExecuteBalances(ctx, dbTx, before)
		if err != nil {
			return nil, false, err
		}
		if executed {
			targetStatus = "done"
		}
	}

	const updateQuery = `
		UPDATE transactions
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND status = 'pending'
		RETURNING id, user_id, amount::text, currency, from_account_id, to_account_id,
		          provider_id, category_id, type, status, description, external_id,
		          occurred_at, created_at, updated_at, deleted_at
	`

	after, err := scanTransaction(dbTx.QueryRowContext(ctx, updateQuery, targetStatus, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if err := dbTx.Commit(); err != nil {
				return nil, false, fmt.Errorf("commit concurrent no-op resolve transaction tx: %w", err)
			}
			return before, false, nil
		}
		return nil, false, fmt.Errorf("update resolved transaction: %w", err)
	}

	messages, err := r.builder.BuildUpdated(before, after)
	if err != nil {
		return nil, false, fmt.Errorf("build transaction resolution outbox events: %w", err)
	}
	if err := r.insertOutboxEvents(ctx, dbTx, messages); err != nil {
		return nil, false, err
	}

	if err := dbTx.Commit(); err != nil {
		return nil, false, fmt.Errorf("commit resolve transaction tx: %w", err)
	}
	return after, true, nil
}

func (r *transactionRepository) getByIDForResolution(
	ctx context.Context,
	tx *sql.Tx,
	id string,
) (*models.Transaction, error) {
	const query = `
		SELECT id, user_id, amount::text, currency, from_account_id, to_account_id,
		       provider_id, category_id, type, status, description, external_id,
		       occurred_at, created_at, updated_at, deleted_at
		FROM transactions
		WHERE id = $1
		FOR UPDATE
	`
	return scanTransaction(tx.QueryRowContext(ctx, query, id))
}

func (r *transactionRepository) tryExecuteBalances(
	ctx context.Context,
	dbTx *sql.Tx,
	tx *models.Transaction,
) (bool, error) {
	if tx == nil {
		return false, nil
	}

	if _, err := dbTx.ExecContext(ctx, "SAVEPOINT transaction_execution"); err != nil {
		return false, fmt.Errorf("create execution savepoint: %w", err)
	}

	executed := cloneTransaction(tx)
	executed.Status = "done"
	err := r.reconcileExecutionBalances(ctx, dbTx, nil, executed)
	if err != nil {
		if _, rollbackErr := dbTx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT transaction_execution"); rollbackErr != nil {
			return false, fmt.Errorf("rollback execution savepoint: %v (original error: %w)", rollbackErr, err)
		}
		if _, releaseErr := dbTx.ExecContext(ctx, "RELEASE SAVEPOINT transaction_execution"); releaseErr != nil {
			return false, fmt.Errorf("release execution savepoint after rollback: %w", releaseErr)
		}
		if isExecutionRejection(err) {
			return false, nil
		}
		return false, err
	}

	if _, err := dbTx.ExecContext(ctx, "RELEASE SAVEPOINT transaction_execution"); err != nil {
		return false, fmt.Errorf("release execution savepoint: %w", err)
	}
	return true, nil
}
