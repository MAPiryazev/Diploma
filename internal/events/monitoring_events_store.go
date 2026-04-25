package events

import (
	"context"
	"fmt"
	"time"

	"github.com/wb-go/wbf/dbpg"
)

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
