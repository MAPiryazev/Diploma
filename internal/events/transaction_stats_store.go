package events

import (
	"context"
	"fmt"
	"time"

	"github.com/wb-go/wbf/dbpg"
)

type TransactionEventStat struct {
	UserID             string
	Currency           string
	StatDate           time.Time
	CreatedCount       int
	UpdatedCount       int
	DeletedCount       int
	StatusChangedCount int
	CreatedAmount      string
	EventTime          time.Time
}

type TransactionEventStatsStore interface {
	Apply(ctx context.Context, stat TransactionEventStat) error
}

type PostgresTransactionEventStatsStore struct {
	db *dbpg.DB
}

func NewPostgresTransactionEventStatsStore(db *dbpg.DB) *PostgresTransactionEventStatsStore {
	return &PostgresTransactionEventStatsStore{db: db}
}

func (s *PostgresTransactionEventStatsStore) Apply(ctx context.Context, stat TransactionEventStat) error {
	if stat.CreatedAmount == "" {
		stat.CreatedAmount = "0"
	}
	if stat.EventTime.IsZero() {
		stat.EventTime = time.Now().UTC()
	}
	if stat.StatDate.IsZero() {
		stat.StatDate = stat.EventTime
	}

	const q = `
		INSERT INTO transaction_event_stats (
			stat_date, user_id, currency,
			created_count, updated_count, deleted_count, status_changed_count,
			created_amount, last_event_time
		) VALUES ($1::date, $2, $3, $4, $5, $6, $7, $8::numeric, $9)
		ON CONFLICT (stat_date, user_id, currency)
		DO UPDATE SET
			created_count = transaction_event_stats.created_count + EXCLUDED.created_count,
			updated_count = transaction_event_stats.updated_count + EXCLUDED.updated_count,
			deleted_count = transaction_event_stats.deleted_count + EXCLUDED.deleted_count,
			status_changed_count = transaction_event_stats.status_changed_count + EXCLUDED.status_changed_count,
			created_amount = transaction_event_stats.created_amount + EXCLUDED.created_amount,
			last_event_time = GREATEST(transaction_event_stats.last_event_time, EXCLUDED.last_event_time),
			updated_at = NOW()
	`
	if _, err := s.db.Master.ExecContext(
		ctx,
		q,
		stat.StatDate,
		stat.UserID,
		stat.Currency,
		stat.CreatedCount,
		stat.UpdatedCount,
		stat.DeletedCount,
		stat.StatusChangedCount,
		stat.CreatedAmount,
		stat.EventTime,
	); err != nil {
		return fmt.Errorf("apply transaction event stats: %w", err)
	}
	return nil
}
