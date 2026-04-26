package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	kafkaProducerMessagesSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_producer_messages_sent_total",
			Help: "Kafka producer messages successfully written (WriteMessages returned nil).",
		},
		[]string{"topic", "event_type"},
	)

	kafkaProducerErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_producer_errors_total",
			Help: "Kafka producer write errors (WriteMessages returned error).",
		},
		[]string{"topic", "event_type"},
	)
)

func RecordKafkaProducerMessageSent(topic, eventType string) {
	kafkaProducerMessagesSent.WithLabelValues(labelOrUnknown(topic), labelOrUnknown(eventType)).Inc()
}

func RecordKafkaProducerError(topic, eventType string) {
	kafkaProducerErrors.WithLabelValues(labelOrUnknown(topic), labelOrUnknown(eventType)).Inc()
}
