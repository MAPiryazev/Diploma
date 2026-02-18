package repository

import (
	"context"
	"fmt"

	"github.com/wb-go/wbf/dbpg"
)

type analyticsRepository struct {
	db *dbpg.DB
}

func NewAnalyticsRepository(db *dbpg.DB) AnalyticsRepository {
	return &analyticsRepository{db: db}
}

func (r *analyticsRepository) GetSum(ctx context.Context, userID string, from, to string) (string, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0)::numeric(18,2)::text
		FROM transactions
		WHERE user_id = $1 AND occurred_at >= $2 AND occurred_at <= $3 AND deleted_at IS NULL
	`

	var sum string
	err := r.db.QueryRowContext(ctx, query, userID, from, to).Scan(&sum)
	if err != nil {
		return "", fmt.Errorf("failed to get sum: %w", err)
	}

	return sum, nil
}

func (r *analyticsRepository) GetAvg(ctx context.Context, userID string, from, to string) (string, error) {
	query := `
		SELECT COALESCE(AVG(amount), 0)::numeric(18,2)::text
		FROM transactions
		WHERE user_id = $1 AND occurred_at >= $2 AND occurred_at <= $3 AND deleted_at IS NULL
	`

	var avg string
	err := r.db.QueryRowContext(ctx, query, userID, from, to).Scan(&avg)
	if err != nil {
		return "", fmt.Errorf("failed to get avg: %w", err)
	}

	return avg, nil
}

func (r *analyticsRepository) GetCount(ctx context.Context, userID string, from, to string) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM transactions
		WHERE user_id = $1 AND occurred_at >= $2 AND occurred_at <= $3 AND deleted_at IS NULL
	`

	var count int64
	err := r.db.QueryRowContext(ctx, query, userID, from, to).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get count: %w", err)
	}

	return count, nil
}

func (r *analyticsRepository) GetMedian(ctx context.Context, userID string, from, to string) (string, error) {
	query := `
		SELECT PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY amount)::numeric(18,2)::text
		FROM transactions
		WHERE user_id = $1 AND occurred_at >= $2 AND occurred_at <= $3 AND deleted_at IS NULL
	`

	var median string
	err := r.db.QueryRowContext(ctx, query, userID, from, to).Scan(&median)
	if err != nil {
		return "", fmt.Errorf("failed to get median: %w", err)
	}

	return median, nil
}

func (r *analyticsRepository) GetPercentile90(ctx context.Context, userID string, from, to string) (string, error) {
	query := `
		SELECT PERCENTILE_CONT(0.9) WITHIN GROUP (ORDER BY amount)::numeric(18,2)::text
		FROM transactions
		WHERE user_id = $1 AND occurred_at >= $2 AND occurred_at <= $3 AND deleted_at IS NULL
	`

	var p90 string
	err := r.db.QueryRowContext(ctx, query, userID, from, to).Scan(&p90)
	if err != nil {
		return "", fmt.Errorf("failed to get percentile 90: %w", err)
	}

	return p90, nil
}
