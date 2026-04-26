package events

import (
	"context"
	"errors"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// NewReplayWriter returns a writer for replaying transaction lifecycle payloads into topic.
func NewReplayWriter(brokers []string, topic string) (*kafka.Writer, error) {
	if len(brokers) == 0 {
		return nil, errors.New("brokers are empty")
	}
	if topic == "" {
		return nil, errors.New("topic is empty")
	}
	return newKafkaWriter(brokers, topic, &kafka.Hash{}), nil
}

// RepublishTransactionCreatedPayload writes the original transaction event JSON back to the main topic.
func RepublishTransactionCreatedPayload(ctx context.Context, w *kafka.Writer, payload []byte) error {
	if w == nil {
		return errors.New("writer is nil")
	}
	if len(payload) == 0 {
		return errors.New("payload is empty")
	}

	env, err := ParseTransactionEventJSON(payload)
	if err != nil {
		return fmt.Errorf("payload is not a valid transaction event envelope: %w", err)
	}

	key, err := userIDKeyFromTransactionEvent(env)
	if err != nil {
		return err
	}

	if err := w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: payload,
	}); err != nil {
		return fmt.Errorf("replay write: %w", err)
	}

	return nil
}

func userIDKeyFromTransactionEvent(env *TransactionEventEnvelope) (string, error) {
	tx := env.Transaction
	if tx == nil {
		tx = env.After
	}
	if tx == nil {
		tx = env.Before
	}
	if tx == nil || tx.UserID == "" {
		return "", errors.New("transaction.user_id missing for routing key")
	}
	return tx.UserID, nil
}
