package events

import (
	"context"
	"fmt"
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/observability"
	"github.com/segmentio/kafka-go"
)

const (
	kafkaWriterBatchSize    = 1
	kafkaWriterBatchTimeout = 10 * time.Millisecond
	kafkaWriterWriteTimeout = 5 * time.Second
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

	w := newKafkaWriter(brokers, topic, &kafka.Hash{})

	return &KafkaPublisher{
		writer: w,
		topic:  topic,
	}, nil
}

func newKafkaWriter(brokers []string, topic string, balancer kafka.Balancer) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		RequiredAcks: kafka.RequireOne,
		Balancer:     balancer,
		BatchSize:    kafkaWriterBatchSize,
		BatchTimeout: kafkaWriterBatchTimeout,
		WriteTimeout: kafkaWriterWriteTimeout,
	}
}

func (p *KafkaPublisher) PublishTransactionCreated(ctx context.Context, tx *models.Transaction) error {
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	payload, err := MarshalTransactionCreatedEnvelope(tx, tx.CreatedAt)
	if err != nil {
		return err
	}

	return p.PublishRaw(ctx, []byte(tx.UserID), payload)
}

func (p *KafkaPublisher) PublishRaw(ctx context.Context, key, payload []byte) error {
	eventType := ""
	if env, err := ParseTransactionEventJSON(payload); err == nil {
		eventType = env.EventType
	}

	msg := kafka.Message{
		Key:   key,
		Value: payload,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		observability.RecordKafkaProducerError(p.topic, eventType)
		return fmt.Errorf("write kafka message to topic %s: %w", p.topic, err)
	}

	observability.RecordKafkaProducerMessageSent(p.topic, eventType)
	return nil
}

func (p *KafkaPublisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
