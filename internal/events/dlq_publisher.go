package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

type DLQPublisher struct {
	writer *kafka.Writer
	topic  string
}

type DLQMessage struct {
	FailedAt      time.Time `json:"failed_at"`
	OriginalTopic string    `json:"original_topic"`
	Partition     int       `json:"partition"`
	Offset        int64     `json:"offset"`
	EventID       string    `json:"event_id,omitempty"`
	EventType     string    `json:"event_type,omitempty"`
	Reason        string    `json:"reason"`
	Payload       string    `json:"payload"`
}

func NewDLQPublisher(brokers []string, topic string) (*DLQPublisher, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are empty")
	}
	if topic == "" {
		return nil, fmt.Errorf("dlq topic is empty")
	}

	writer := newKafkaWriter(brokers, topic, &kafka.LeastBytes{})

	return &DLQPublisher{
		writer: writer,
		topic:  topic,
	}, nil
}

func (p *DLQPublisher) Publish(ctx context.Context, msg DLQMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal dlq message: %w", err)
	}

	if err := p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.EventID),
		Value: payload,
		Time:  msg.FailedAt,
	}); err != nil {
		return fmt.Errorf("publish dlq message to %s: %w", p.topic, err)
	}

	return nil
}

func (p *DLQPublisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
