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

type accountRepository struct {
	db *dbpg.DB
}

func NewAccountRepository(db *dbpg.DB) AccountRepository {
	return &accountRepository{db: db}
}

func (r *accountRepository) Create(ctx context.Context, acc *models.Account) (*models.Account, error) {
	query := `
		INSERT INTO accounts (user_id, name, number)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	var id string
	var createdAt time.Time
	err := r.db.Master.QueryRowContext(ctx, query, acc.UserID, acc.Name, acc.Number).
		Scan(&id, &createdAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	acc.ID = id
	acc.CreatedAt = createdAt
	return acc, nil
}

func (r *accountRepository) GetByID(ctx context.Context, id, userID string) (*models.Account, error) {
	query := `
		SELECT id, user_id, name, number, created_at
		FROM accounts
		WHERE id = $1 AND user_id = $2
	`

	var acc models.Account
	err := r.db.QueryRowContext(ctx, query, id, userID).
		Scan(&acc.ID, &acc.UserID, &acc.Name, &acc.Number, &acc.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return &acc, nil
}

func (r *accountRepository) ListByUser(ctx context.Context, userID string) ([]*models.Account, error) {
	query := `
		SELECT id, user_id, name, number, created_at
		FROM accounts
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []*models.Account
	for rows.Next() {
		var acc models.Account
		err := rows.Scan(&acc.ID, &acc.UserID, &acc.Name, &acc.Number, &acc.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		accounts = append(accounts, &acc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return accounts, nil
}
