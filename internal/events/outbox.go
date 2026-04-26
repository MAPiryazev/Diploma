package events

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/observability"
	"github.com/wb-go/wbf/dbpg"
)

const (
	defaultOutboxBatchSize       = 25
	defaultOutboxPollInterval    = 500 * time.Millisecond
	defaultOutboxPublishTimeout  = 5 * time.Second
	defaultOutboxRetryDelay      = 2 * time.Second
	defaultOutboxMaxErrorMessage = 1000
)

type OutboxMessage struct {
	ID        string
	EventID   string
	EventType string
	Topic     string
	Key       string
	Payload   []byte
}

type OutboxStore interface {
	FetchReady(ctx context.Context, limit int) ([]OutboxMessage, error)
	MarkPublished(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string, publishErr error, retryDelay time.Duration) error
}

type RawPublisher interface {
	PublishRaw(ctx context.Context, key, payload []byte) error
}

type PostgresOutboxStore struct {
	db *dbpg.DB
}

func NewPostgresOutboxStore(db *dbpg.DB) *PostgresOutboxStore {
	return &PostgresOutboxStore{db: db}
}

func (s *PostgresOutboxStore) FetchReady(ctx context.Context, limit int) ([]OutboxMessage, error) {
	if limit <= 0 {
		limit = defaultOutboxBatchSize
	}

	tx, err := s.db.Master.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin outbox fetch tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	const selectQuery = `
		SELECT id, event_id, event_type, topic, message_key, payload
		FROM event_outbox
		WHERE
			(status IN ('pending', 'failed') AND next_attempt_at <= NOW())
			OR (status = 'processing' AND locked_at < NOW() - INTERVAL '30 seconds')
		ORDER BY created_at
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`
	rows, err := tx.QueryContext(ctx, selectQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("select ready outbox events: %w", err)
	}
	defer rows.Close()

	messages := make([]OutboxMessage, 0, limit)
	for rows.Next() {
		var msg OutboxMessage
		if err := rows.Scan(&msg.ID, &msg.EventID, &msg.EventType, &msg.Topic, &msg.Key, &msg.Payload); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("outbox rows: %w", err)
	}

	const lockQuery = `
		UPDATE event_outbox
		SET status = 'processing', attempts = attempts + 1, locked_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	for _, msg := range messages {
		if _, err := tx.ExecContext(ctx, lockQuery, msg.ID); err != nil {
			return nil, fmt.Errorf("lock outbox event %s: %w", msg.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit outbox fetch tx: %w", err)
	}
	return messages, nil
}

func (s *PostgresOutboxStore) MarkPublished(ctx context.Context, id string) error {
	const q = `
		UPDATE event_outbox
		SET status = 'sent', published_at = NOW(), last_error = NULL, updated_at = NOW()
		WHERE id = $1
	`
	if _, err := s.db.Master.ExecContext(ctx, q, id); err != nil {
		return fmt.Errorf("mark outbox event published: %w", err)
	}
	return nil
}

func (s *PostgresOutboxStore) MarkFailed(ctx context.Context, id string, publishErr error, retryDelay time.Duration) error {
	if retryDelay <= 0 {
		retryDelay = defaultOutboxRetryDelay
	}
	message := ""
	if publishErr != nil {
		message = publishErr.Error()
	}
	if len(message) > defaultOutboxMaxErrorMessage {
		message = message[:defaultOutboxMaxErrorMessage]
	}

	const q = `
		UPDATE event_outbox
		SET status = 'failed', last_error = $2, next_attempt_at = $3, updated_at = NOW()
		WHERE id = $1
	`
	if _, err := s.db.Master.ExecContext(ctx, q, id, message, time.Now().UTC().Add(retryDelay)); err != nil {
		return fmt.Errorf("mark outbox event failed: %w", err)
	}
	return nil
}

type OutboxRelayConfig struct {
	BatchSize      int
	PollInterval   time.Duration
	PublishTimeout time.Duration
	RetryDelay     time.Duration
}

type OutboxRelay struct {
	store     OutboxStore
	publisher RawPublisher
	cfg       OutboxRelayConfig
}

func NewOutboxRelay(store OutboxStore, publisher RawPublisher, cfg OutboxRelayConfig) *OutboxRelay {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = defaultOutboxBatchSize
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaultOutboxPollInterval
	}
	if cfg.PublishTimeout <= 0 {
		cfg.PublishTimeout = defaultOutboxPublishTimeout
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = defaultOutboxRetryDelay
	}
	return &OutboxRelay{store: store, publisher: publisher, cfg: cfg}
}

func (r *OutboxRelay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.cfg.PollInterval)
	defer ticker.Stop()

	for {
		r.publishReady(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *OutboxRelay) publishReady(ctx context.Context) {
	for {
		messages, err := r.store.FetchReady(ctx, r.cfg.BatchSize)
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("outbox fetch failed: %v", err)
			}
			return
		}
		if len(messages) == 0 {
			return
		}

		for _, msg := range messages {
			start := time.Now()
			publishCtx, cancel := context.WithTimeout(ctx, r.cfg.PublishTimeout)
			err := r.publisher.PublishRaw(publishCtx, []byte(msg.Key), msg.Payload)
			cancel()

			if err != nil {
				log.Printf("outbox publish failed: outbox_id=%s event_id=%s topic=%s err=%v", msg.ID, msg.EventID, msg.Topic, err)
				observability.RecordOutboxPublishFailed(msg.EventType)
				if markErr := r.store.MarkFailed(ctx, msg.ID, err, r.cfg.RetryDelay); markErr != nil && ctx.Err() == nil {
					log.Printf("outbox mark failed error: outbox_id=%s err=%v", msg.ID, markErr)
				}
				continue
			}

			if err := r.store.MarkPublished(ctx, msg.ID); err != nil && ctx.Err() == nil {
				log.Printf("outbox mark published failed: outbox_id=%s event_id=%s err=%v", msg.ID, msg.EventID, err)
				observability.RecordOutboxPublishFailed(msg.EventType)
				continue
			}
			observability.RecordOutboxPublished(msg.EventType)
			observability.ObserveOutboxPublishDuration(msg.EventType, time.Since(start))
		}
	}
}
