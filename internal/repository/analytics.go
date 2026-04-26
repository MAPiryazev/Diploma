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

func (r *analyticsRepository) GetStreamStats(ctx context.Context, userID string, from, to string) ([]StreamAnalyticsRow, error) {
	const query = `
		SELECT
			stat_date::text,
			currency,
			created_count,
			updated_count,
			deleted_count,
			status_changed_count,
			created_amount::numeric(18,2)::text,
			last_event_time::text
		FROM transaction_event_stats
		WHERE user_id = $1 AND stat_date >= $2::date AND stat_date <= $3::date
		ORDER BY stat_date, currency
	`

	rows, err := r.db.QueryContext(ctx, query, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream stats: %w", err)
	}
	defer rows.Close()

	stats := make([]StreamAnalyticsRow, 0)
	for rows.Next() {
		var row StreamAnalyticsRow
		if err := rows.Scan(
			&row.StatDate,
			&row.Currency,
			&row.CreatedCount,
			&row.UpdatedCount,
			&row.DeletedCount,
			&row.StatusChangedCount,
			&row.CreatedAmount,
			&row.LastEventTime,
		); err != nil {
			return nil, fmt.Errorf("scan stream stats: %w", err)
		}
		stats = append(stats, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("stream stats rows: %w", err)
	}

	return stats, nil
}
