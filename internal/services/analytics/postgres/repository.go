package postgres

import (
	"context"
	"fmt"
	"strings"

	analyticsapp "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/analytics/app"
	"github.com/wb-go/wbf/dbpg"
)

type StreamAnalyticsRow struct {
	StatDate           string
	Currency           string
	CreatedCount       int64
	UpdatedCount       int64
	DeletedCount       int64
	StatusChangedCount int64
	CreatedAmount      string
	LastEventTime      string
}

type Repository struct {
	db *dbpg.DB
}

func NewRepository(db *dbpg.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetSum(ctx context.Context, userID string, from, to string) (string, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0)::numeric(18,2)::text
		FROM analytics_transactions
		WHERE user_id = $1 AND occurred_at >= $2 AND occurred_at <= $3 AND deleted_at IS NULL
	`

	var sum string
	if err := r.db.QueryRowContext(ctx, query, userID, from, to).Scan(&sum); err != nil {
		return "", fmt.Errorf("failed to get sum: %w", err)
	}
	return sum, nil
}

func (r *Repository) GetAvg(ctx context.Context, userID string, from, to string) (string, error) {
	query := `
		SELECT COALESCE(AVG(amount), 0)::numeric(18,2)::text
		FROM analytics_transactions
		WHERE user_id = $1 AND occurred_at >= $2 AND occurred_at <= $3 AND deleted_at IS NULL
	`

	var avg string
	if err := r.db.QueryRowContext(ctx, query, userID, from, to).Scan(&avg); err != nil {
		return "", fmt.Errorf("failed to get avg: %w", err)
	}
	return avg, nil
}

func (r *Repository) GetCount(ctx context.Context, userID string, from, to string) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM analytics_transactions
		WHERE user_id = $1 AND occurred_at >= $2 AND occurred_at <= $3 AND deleted_at IS NULL
	`

	var count int64
	if err := r.db.QueryRowContext(ctx, query, userID, from, to).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to get count: %w", err)
	}
	return count, nil
}

func (r *Repository) GetMedian(ctx context.Context, userID string, from, to string) (string, error) {
	query := `
		SELECT COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY amount), 0)::numeric(18,2)::text
		FROM analytics_transactions
		WHERE user_id = $1 AND occurred_at >= $2 AND occurred_at <= $3 AND deleted_at IS NULL
	`

	var median string
	if err := r.db.QueryRowContext(ctx, query, userID, from, to).Scan(&median); err != nil {
		return "", fmt.Errorf("failed to get median: %w", err)
	}
	return median, nil
}

func (r *Repository) GetPercentile90(ctx context.Context, userID string, from, to string) (string, error) {
	query := `
		SELECT COALESCE(PERCENTILE_CONT(0.9) WITHIN GROUP (ORDER BY amount), 0)::numeric(18,2)::text
		FROM analytics_transactions
		WHERE user_id = $1 AND occurred_at >= $2 AND occurred_at <= $3 AND deleted_at IS NULL
	`

	var p90 string
	if err := r.db.QueryRowContext(ctx, query, userID, from, to).Scan(&p90); err != nil {
		return "", fmt.Errorf("failed to get percentile 90: %w", err)
	}
	return p90, nil
}

func (r *Repository) GetStreamStats(ctx context.Context, userID string, from, to string) ([]StreamAnalyticsRow, error) {
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

func (r *Repository) ListMonitoringAlerts(
	ctx context.Context,
	userID string,
	filter analyticsapp.MonitoringAlertsFilter,
) ([]analyticsapp.MonitoringAlert, error) {
	args := []any{userID}
	query := `
		SELECT
			id::text,
			transaction_id::text,
			user_id::text,
			rule_code,
			severity,
			reason,
			event_time::text,
			created_at::text
		FROM monitoring_events
		WHERE user_id = $1
	`

	if filter.Severity != "" {
		args = append(args, filter.Severity)
		query += fmt.Sprintf(" AND severity = $%d", len(args))
	}
	if filter.RuleCode != "" {
		args = append(args, filter.RuleCode)
		query += fmt.Sprintf(" AND rule_code = $%d", len(args))
	}
	if strings.TrimSpace(filter.From) != "" {
		args = append(args, filter.From)
		query += fmt.Sprintf(" AND event_time >= $%d::timestamptz", len(args))
	}
	if strings.TrimSpace(filter.To) != "" {
		args = append(args, filter.To)
		query += fmt.Sprintf(" AND event_time <= $%d::timestamptz", len(args))
	}

	args = append(args, filter.Limit)
	query += fmt.Sprintf(" ORDER BY event_time DESC, created_at DESC LIMIT $%d", len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list monitoring alerts: %w", err)
	}
	defer rows.Close()

	alerts := make([]analyticsapp.MonitoringAlert, 0)
	for rows.Next() {
		var alert analyticsapp.MonitoringAlert
		if err := rows.Scan(
			&alert.ID,
			&alert.TransactionID,
			&alert.UserID,
			&alert.RuleCode,
			&alert.Severity,
			&alert.Reason,
			&alert.EventTime,
			&alert.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan monitoring alert: %w", err)
		}
		alerts = append(alerts, alert)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("monitoring alerts rows: %w", err)
	}

	return alerts, nil
}
