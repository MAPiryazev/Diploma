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

type categoryRepository struct {
	db *dbpg.DB
}

func NewCategoryRepository(db *dbpg.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) Create(ctx context.Context, cat *models.Category) (*models.Category, error) {
	query := `
		INSERT INTO categories (user_id, name)
		VALUES ($1, $2)
		RETURNING id, created_at
	`

	var id string
	var createdAt time.Time
	err := r.db.Master.QueryRowContext(ctx, query, cat.UserID, cat.Name).
		Scan(&id, &createdAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	cat.ID = id
	cat.CreatedAt = createdAt
	return cat, nil
}

func (r *categoryRepository) GetByID(ctx context.Context, id, userID string) (*models.Category, error) {
	query := `
		SELECT id, user_id, name, created_at
		FROM categories
		WHERE id = $1 AND user_id = $2
	`

	var cat models.Category
	err := r.db.QueryRowContext(ctx, query, id, userID).
		Scan(&cat.ID, &cat.UserID, &cat.Name, &cat.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	return &cat, nil
}

func (r *categoryRepository) ListByUser(ctx context.Context, userID string) ([]*models.Category, error) {
	query := `
		SELECT id, user_id, name, created_at
		FROM categories
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	defer rows.Close()

	var categories []*models.Category
	for rows.Next() {
		var cat models.Category
		err := rows.Scan(&cat.ID, &cat.UserID, &cat.Name, &cat.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, &cat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return categories, nil
}
