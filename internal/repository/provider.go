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

type providerRepository struct {
	db *dbpg.DB
}

func NewProviderRepository(db *dbpg.DB) ProviderRepository {
	return &providerRepository{db: db}
}

func (r *providerRepository) Create(ctx context.Context, provider *models.Provider) (*models.Provider, error) {
	query := `
		INSERT INTO providers (name)
		VALUES ($1)
		RETURNING id, created_at
	`

	var id string
	var createdAt time.Time
	err := r.db.Master.QueryRowContext(ctx, query, provider.Name).
		Scan(&id, &createdAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	provider.ID = id
	provider.CreatedAt = createdAt
	return provider, nil
}

func (r *providerRepository) GetByID(ctx context.Context, id string) (*models.Provider, error) {
	query := `SELECT id, name, created_at FROM providers WHERE id = $1`

	var provider models.Provider
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&provider.ID, &provider.Name, &provider.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	return &provider, nil
}

func (r *providerRepository) GetByName(ctx context.Context, name string) (*models.Provider, error) {
	query := `SELECT id, name, created_at FROM providers WHERE name = $1`

	var provider models.Provider
	err := r.db.QueryRowContext(ctx, query, name).
		Scan(&provider.ID, &provider.Name, &provider.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	return &provider, nil
}
