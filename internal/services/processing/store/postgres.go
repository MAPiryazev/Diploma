package store

import (
	"context"
	"fmt"
	"time"

	"github.com/wb-go/wbf/dbpg"
)

type ProcessedEventsStore interface {
	SaveIfNew(ctx context.Context, subscriberName, eventID, eventType string) (bool, error)
}

type MonitoringEvent struct {
	TransactionID string
	UserID        string
	RuleCode      string
	Severity      string
	Reason        string
	EventTime     time.Time
}

type MonitoringEventsStore interface {
	Save(ctx context.Context, event MonitoringEvent) error
}

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

type AnalyticsTransaction struct {
	TransactionID string
	UserID        string
	Amount        string
	Currency      string
	FromAccountID *string
	ToAccountID   *string
	ProviderID    *string
	CategoryID    *string
	Type          string
	Status        string
	Description   *string
	ExternalID    *string
	OccurredAt    time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

type AnalyticsTransactionsStore interface {
	Apply(ctx context.Context, tx AnalyticsTransaction) error
}

type PostgresProcessedEventsStore struct {
	db *dbpg.DB
}

func NewPostgresProcessedEventsStore(db *dbpg.DB) *PostgresProcessedEventsStore {
	return &PostgresProcessedEventsStore{db: db}
}

func (s *PostgresProcessedEventsStore) SaveIfNew(ctx context.Context, subscriberName, eventID, eventType string) (bool, error) {
	const q = `
		INSERT INTO processed_events(subscriber_name, event_id, event_type)
		VALUES ($1, $2, $3)
		ON CONFLICT (subscriber_name, event_id) DO NOTHING
	`

	res, err := s.db.Master.ExecContext(ctx, q, subscriberName, eventID, eventType)
	if err != nil {
		return false, fmt.Errorf("insert processed_events: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("processed_events rows affected: %w", err)
	}

	return affected > 0, nil
}

type PostgresMonitoringEventsStore struct {
	db *dbpg.DB
}

func NewPostgresMonitoringEventsStore(db *dbpg.DB) *PostgresMonitoringEventsStore {
	return &PostgresMonitoringEventsStore{db: db}
}

func (s *PostgresMonitoringEventsStore) Save(ctx context.Context, event MonitoringEvent) error {
	const q = `
		INSERT INTO monitoring_events (
			transaction_id, user_id, rule_code, severity, reason, event_time
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (rule_code, transaction_id) DO NOTHING
	`

	if _, err := s.db.Master.ExecContext(
		ctx,
		q,
		event.TransactionID,
		event.UserID,
		event.RuleCode,
		event.Severity,
		event.Reason,
		event.EventTime,
	); err != nil {
		return fmt.Errorf("insert monitoring event: %w", err)
	}
	return nil
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

type PostgresAnalyticsTransactionsStore struct {
	db *dbpg.DB
}

func NewPostgresAnalyticsTransactionsStore(db *dbpg.DB) *PostgresAnalyticsTransactionsStore {
	return &PostgresAnalyticsTransactionsStore{db: db}
}

func (s *PostgresAnalyticsTransactionsStore) Apply(ctx context.Context, tx AnalyticsTransaction) error {
	const q = `
		INSERT INTO analytics_transactions (
			transaction_id, user_id, amount, currency,
			from_account_id, to_account_id, provider_id, category_id,
			type, status, description, external_id,
			occurred_at, created_at, updated_at, deleted_at
		) VALUES (
			$1, $2, $3::numeric, $4,
			$5, $6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15, $16
		)
		ON CONFLICT (transaction_id)
		DO UPDATE SET
			user_id = EXCLUDED.user_id,
			amount = EXCLUDED.amount,
			currency = EXCLUDED.currency,
			from_account_id = EXCLUDED.from_account_id,
			to_account_id = EXCLUDED.to_account_id,
			provider_id = EXCLUDED.provider_id,
			category_id = EXCLUDED.category_id,
			type = EXCLUDED.type,
			status = EXCLUDED.status,
			description = EXCLUDED.description,
			external_id = EXCLUDED.external_id,
			occurred_at = EXCLUDED.occurred_at,
			created_at = EXCLUDED.created_at,
			updated_at = EXCLUDED.updated_at,
			deleted_at = EXCLUDED.deleted_at
	`

	if _, err := s.db.Master.ExecContext(
		ctx,
		q,
		tx.TransactionID,
		tx.UserID,
		tx.Amount,
		tx.Currency,
		nullableString(tx.FromAccountID),
		nullableString(tx.ToAccountID),
		nullableString(tx.ProviderID),
		nullableString(tx.CategoryID),
		tx.Type,
		tx.Status,
		nullableString(tx.Description),
		nullableString(tx.ExternalID),
		tx.OccurredAt,
		tx.CreatedAt,
		tx.UpdatedAt,
		nullableTime(tx.DeletedAt),
	); err != nil {
		return fmt.Errorf("apply analytics transaction projection: %w", err)
	}
	return nil
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}
