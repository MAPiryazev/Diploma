package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// NewReplayWriter returns a writer for replaying transaction.created payloads into topic (same routing as KafkaPublisher).
func NewReplayWriter(brokers []string, topic string) (*kafka.Writer, error) {
	if len(brokers) == 0 {
		return nil, errors.New("brokers are empty")
	}
	if topic == "" {
		return nil, errors.New("topic is empty")
	}
	return &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		RequiredAcks: kafka.RequireOne,
		Balancer:     &kafka.Hash{},
	}, nil
}

// RepublishTransactionCreatedPayload writes the original transaction.created JSON back to the main topic
// (same key strategy as KafkaPublisher: partition by user_id).
func RepublishTransactionCreatedPayload(ctx context.Context, w *kafka.Writer, payload []byte) error {
	if w == nil {
		return errors.New("writer is nil")
	}
	if len(payload) == 0 {
		return errors.New("payload is empty")
	}

	if _, err := ParseTransactionCreatedJSON(payload); err != nil {
		return fmt.Errorf("payload is not a valid transaction.created envelope: %w", err)
	}

	key, err := userIDKeyFromTransactionCreatedJSON(payload)
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

func userIDKeyFromTransactionCreatedJSON(payload []byte) (string, error) {
	var v struct {
		Transaction *struct {
			UserID string `json:"user_id"`
		} `json:"transaction"`
	}
	if err := json.Unmarshal(payload, &v); err != nil {
		return "", fmt.Errorf("unmarshal for routing key: %w", err)
	}
	if v.Transaction == nil || v.Transaction.UserID == "" {
		return "", errors.New("transaction.user_id missing for routing key")
	}
	return v.Transaction.UserID, nil
}
