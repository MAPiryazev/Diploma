package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	apperrors "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/errors"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
	"github.com/wb-go/wbf/dbpg"
)

type transactionRepository struct {
	db *dbpg.DB
}

func NewTransactionRepository(db *dbpg.DB) TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) Create(ctx context.Context, tx *models.Transaction) (*models.Transaction, error) {
	query := `
		INSERT INTO transactions (
			user_id, amount, currency, from_account_id, to_account_id,
			provider_id, category_id, type, status, description,
			external_id, occurred_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`

	var id string
	var createdAt, updatedAt time.Time

	err := r.db.Master.QueryRowContext(
		ctx,
		query,
		tx.UserID, tx.Amount, tx.Currency, tx.FromAccountID, tx.ToAccountID,
		tx.ProviderID, tx.CategoryID, tx.Type, tx.Status, tx.Description,
		tx.ExternalID, tx.OccurredAt,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	tx.ID = id
	tx.CreatedAt = createdAt
	tx.UpdatedAt = updatedAt
	return tx, nil
}

func (r *transactionRepository) GetByID(ctx context.Context, id, userID string) (*models.Transaction, error) {
	query := `
		SELECT id, user_id, amount, currency, from_account_id, to_account_id,
		       provider_id, category_id, type, status, description, external_id,
		       occurred_at, created_at, updated_at, deleted_at
		FROM transactions
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`

	var tx models.Transaction
	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(
		&tx.ID, &tx.UserID, &tx.Amount, &tx.Currency,
		&tx.FromAccountID, &tx.ToAccountID, &tx.ProviderID, &tx.CategoryID,
		&tx.Type, &tx.Status, &tx.Description, &tx.ExternalID,
		&tx.OccurredAt, &tx.CreatedAt, &tx.UpdatedAt, &tx.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &tx, nil
}

func (r *transactionRepository) ListByUser(ctx context.Context, userID string) ([]*models.Transaction, error) {
	query := `
		SELECT id, user_id, amount, currency, from_account_id, to_account_id,
		       provider_id, category_id, type, status, description, external_id,
		       occurred_at, created_at, updated_at, deleted_at
		FROM transactions
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY occurred_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	defer rows.Close()

	var txs []*models.Transaction
	for rows.Next() {
		var tx models.Transaction
		err := rows.Scan(
			&tx.ID, &tx.UserID, &tx.Amount, &tx.Currency,
			&tx.FromAccountID, &tx.ToAccountID, &tx.ProviderID, &tx.CategoryID,
			&tx.Type, &tx.Status, &tx.Description, &tx.ExternalID,
			&tx.OccurredAt, &tx.CreatedAt, &tx.UpdatedAt, &tx.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		txs = append(txs, &tx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return txs, nil
}

func (r *transactionRepository) Update(ctx context.Context, tx *models.Transaction) error {
	query := `
		UPDATE transactions
		SET amount = $1, currency = $2, from_account_id = $3, to_account_id = $4,
		    provider_id = $5, category_id = $6, type = $7, status = $8,
		    description = $9, external_id = $10, occurred_at = $11, updated_at = NOW()
		WHERE id = $12 AND user_id = $13 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		tx.Amount, tx.Currency, tx.FromAccountID, tx.ToAccountID,
		tx.ProviderID, tx.CategoryID, tx.Type, tx.Status,
		tx.Description, tx.ExternalID, tx.OccurredAt, tx.ID, tx.UserID,
	)

	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}

func (r *transactionRepository) Delete(ctx context.Context, id, userID string) error {
	query := `
		UPDATE transactions
		SET deleted_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}
