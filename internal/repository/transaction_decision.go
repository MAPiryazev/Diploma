package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
	transactionevents "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/contracts/transactionevents"
	"github.com/wb-go/wbf/dbpg"
)

type TransactionDecisionEventBuilder interface {
	BuildApproved(tx *models.Transaction) ([]OutboxMessage, error)
	BuildRejected(tx *models.Transaction) ([]OutboxMessage, error)
}

type transactionDecisionPublisher struct {
	db      *dbpg.DB
	topic   string
	builder TransactionDecisionEventBuilder
}

func NewTransactionDecisionPublisher(
	db *dbpg.DB,
	builder TransactionDecisionEventBuilder,
	topic ...string,
) TransactionDecisionPublisher {
	kafkaTopic := transactionevents.DefaultTransactionsTopic
	if len(topic) > 0 && strings.TrimSpace(topic[0]) != "" {
		kafkaTopic = strings.TrimSpace(topic[0])
	}
	return &transactionDecisionPublisher{db: db, topic: kafkaTopic, builder: builder}
}

func (p *transactionDecisionPublisher) PublishDecision(
	ctx context.Context,
	tx *models.Transaction,
	approved bool,
) error {
	if p == nil {
		return fmt.Errorf("transaction decision publisher is nil")
	}
	if p.db == nil {
		return fmt.Errorf("transaction decision publisher db is nil")
	}
	if p.builder == nil {
		return fmt.Errorf("transaction decision publisher builder is nil")
	}
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	dbTx, err := p.db.Master.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin publish decision tx: %w", err)
	}
	defer func() { _ = dbTx.Rollback() }()

	var messages []OutboxMessage
	if approved {
		messages, err = p.builder.BuildApproved(tx)
	} else {
		messages, err = p.builder.BuildRejected(tx)
	}
	if err != nil {
		return fmt.Errorf("build transaction decision outbox event: %w", err)
	}

	const outboxQuery = `
		INSERT INTO event_outbox (event_id, aggregate_id, event_type, topic, message_key, payload)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	for _, message := range messages {
		if _, err := dbTx.ExecContext(
			ctx,
			outboxQuery,
			message.EventID,
			message.AggregateID,
			message.EventType,
			p.topic,
			message.MessageKey,
			string(message.Payload),
		); err != nil {
			return fmt.Errorf("failed to create decision outbox event: %w", err)
		}
	}

	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit publish decision tx: %w", err)
	}
	return nil
}
