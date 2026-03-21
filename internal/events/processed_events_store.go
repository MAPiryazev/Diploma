package events

import (
	"context"
	"fmt"

	"github.com/wb-go/wbf/dbpg"
)

type ProcessedEventsStore interface {
	SaveIfNew(ctx context.Context, eventID, eventType string) (bool, error)
}

type PostgresProcessedEventsStore struct {
	db *dbpg.DB
}

func NewPostgresProcessedEventsStore(db *dbpg.DB) *PostgresProcessedEventsStore {
	return &PostgresProcessedEventsStore{db: db}
}

func (s *PostgresProcessedEventsStore) SaveIfNew(ctx context.Context, eventID, eventType string) (bool, error) {
	const q = `
		INSERT INTO processed_events(event_id, event_type)
		VALUES ($1, $2)
		ON CONFLICT (event_id) DO NOTHING
	`

	res, err := s.db.Master.ExecContext(ctx, q, eventID, eventType)
	if err != nil {
		return false, fmt.Errorf("insert processed_events: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("processed_events rows affected: %w", err)
	}

	return affected > 0, nil
}
