package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/wb-go/wbf/dbpg"
)

type riskRepository interface {
	LoadRuleConfigs(ctx context.Context) (map[string]ruleConfig, error)
	CountTransactionsSince(ctx context.Context, userID, excludeTransactionID string, from, to time.Time) (int, error)
	SumTransactionsSince(ctx context.Context, userID, excludeTransactionID string, from, to time.Time) (float64, error)
	CountTransactionsByAmountSince(ctx context.Context, userID, excludeTransactionID, amount string, from, to time.Time) (int, error)
}

type postgresRiskRepository struct {
	db *dbpg.DB
}

func newPostgresRiskRepository(db *dbpg.DB) *postgresRiskRepository {
	return &postgresRiskRepository{db: db}
}

func (r *postgresRiskRepository) LoadRuleConfigs(ctx context.Context) (map[string]ruleConfig, error) {
	const query = `
		SELECT rule_code, enabled, severity, params, version, updated_at
		FROM risk_rules
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("load risk rules: %w", err)
	}
	defer rows.Close()

	configs := make(map[string]ruleConfig)
	for rows.Next() {
		var cfg ruleConfig
		var rawParams []byte
		if err := rows.Scan(
			&cfg.RuleCode,
			&cfg.Enabled,
			&cfg.Severity,
			&rawParams,
			&cfg.Version,
			&cfg.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan risk rule: %w", err)
		}
		if len(rawParams) > 0 {
			if err := json.Unmarshal(rawParams, &cfg.Parameters); err != nil {
				return nil, fmt.Errorf("decode risk rule params for %s: %w", cfg.RuleCode, err)
			}
		}
		configs[cfg.RuleCode] = cfg
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("risk rules rows: %w", err)
	}

	return configs, nil
}

func (r *postgresRiskRepository) CountTransactionsSince(
	ctx context.Context,
	userID, excludeTransactionID string,
	from, to time.Time,
) (int, error) {
	const query = `
		SELECT COUNT(*)
		FROM analytics_transactions
		WHERE user_id = $1
		  AND transaction_id <> $2
		  AND occurred_at >= $3
		  AND occurred_at <= $4
		  AND deleted_at IS NULL
	`

	var count int
	if err := r.db.QueryRowContext(ctx, query, userID, excludeTransactionID, from, to).Scan(&count); err != nil {
		return 0, fmt.Errorf("count transactions since: %w", err)
	}
	return count, nil
}

func (r *postgresRiskRepository) SumTransactionsSince(
	ctx context.Context,
	userID, excludeTransactionID string,
	from, to time.Time,
) (float64, error) {
	const query = `
		SELECT COALESCE(SUM(amount), 0)::numeric(18,2)::text
		FROM analytics_transactions
		WHERE user_id = $1
		  AND transaction_id <> $2
		  AND occurred_at >= $3
		  AND occurred_at <= $4
		  AND deleted_at IS NULL
	`

	var total string
	if err := r.db.QueryRowContext(ctx, query, userID, excludeTransactionID, from, to).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum transactions since: %w", err)
	}

	value, err := strconv.ParseFloat(strings.TrimSpace(total), 64)
	if err != nil {
		return 0, fmt.Errorf("parse summed amount: %w", err)
	}
	return value, nil
}

func (r *postgresRiskRepository) CountTransactionsByAmountSince(
	ctx context.Context,
	userID, excludeTransactionID, amount string,
	from, to time.Time,
) (int, error) {
	const query = `
		SELECT COUNT(*)
		FROM analytics_transactions
		WHERE user_id = $1
		  AND transaction_id <> $2
		  AND amount = $3::numeric
		  AND occurred_at >= $4
		  AND occurred_at <= $5
		  AND deleted_at IS NULL
	`

	var count int
	if err := r.db.QueryRowContext(ctx, query, userID, excludeTransactionID, amount, from, to).Scan(&count); err != nil {
		return 0, fmt.Errorf("count transactions by amount since: %w", err)
	}
	return count, nil
}
