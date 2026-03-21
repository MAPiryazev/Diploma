package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/observability"
	"github.com/segmentio/kafka-go"
)

type KafkaPublisher struct {
	writer *kafka.Writer
	topic  string
}

func NewKafkaPublisher(brokers []string, topic string) (*KafkaPublisher, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are empty")
	}
	if topic == "" {
		return nil, fmt.Errorf("kafka topic is empty")
	}

	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		RequiredAcks: kafka.RequireOne,
		Balancer:     &kafka.Hash{},
	}

	return &KafkaPublisher{
		writer: w,
		topic:  topic,
	}, nil
}

func (p *KafkaPublisher) PublishTransactionCreated(ctx context.Context, tx *models.Transaction) error {
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	event := TransactionCreatedEnvelope{
		EventID:       tx.ID,
		EventType:     EventTypeTransactionCreated,
		EventTime:     time.Now().UTC(),
		CorrelationID: tx.ID,
		SchemaVersion: SupportedSchemaVersion,
		Source:        "diploma-app",
		Transaction:   tx,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal transaction.created event: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(tx.UserID),
		Value: payload,
		Time:  event.EventTime,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		observability.RecordKafkaProducerError(p.topic)
		return fmt.Errorf("write kafka message to topic %s: %w", p.topic, err)
	}

	observability.RecordKafkaProducerMessageSent(p.topic)
	return nil
}

func (p *KafkaPublisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
