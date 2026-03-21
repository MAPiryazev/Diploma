package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	kafkaConsumerMessagesProcessed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_consumer_messages_processed_total",
			Help: "Kafka consumer messages successfully processed and committed.",
		},
	)

	kafkaConsumerMessagesInvalid = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_consumer_messages_invalid_total",
			Help: "Kafka consumer messages rejected by JSON/contract validation.",
		},
	)

	kafkaConsumerMessagesDuplicate = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_consumer_messages_duplicate_total",
			Help: "Kafka messages skipped by deduplication (already processed event_id).",
		},
	)

	kafkaConsumerCommitErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_consumer_commit_errors_total",
			Help: "Kafka consumer failed to commit offsets after processing.",
		},
	)

	kafkaConsumerDLQPublished = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_consumer_dlq_published_total",
			Help: "Kafka messages published to DLQ topic.",
		},
	)

	kafkaConsumerDLQErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_consumer_dlq_errors_total",
			Help: "Kafka consumer failed to publish message to DLQ topic.",
		},
	)

	kafkaConsumerHandleDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "kafka_consumer_handle_duration_seconds",
			Help:    "Time spent handling one Kafka message after validation.",
			Buckets: prometheus.DefBuckets,
		},
	)

	kafkaConsumerHandlerRetries = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "kafka_consumer_handler_retries_total",
			Help: "Extra handler attempts after the first failure (step 6: backoff retry before DLQ).",
		},
	)
)

func RecordKafkaConsumerProcessed() {
	kafkaConsumerMessagesProcessed.Inc()
}

func RecordKafkaConsumerInvalid() {
	kafkaConsumerMessagesInvalid.Inc()
}

func RecordKafkaConsumerDuplicate() {
	kafkaConsumerMessagesDuplicate.Inc()
}

func RecordKafkaConsumerCommitError() {
	kafkaConsumerCommitErrors.Inc()
}

func RecordKafkaConsumerDLQPublished() {
	kafkaConsumerDLQPublished.Inc()
}

func RecordKafkaConsumerDLQError() {
	kafkaConsumerDLQErrors.Inc()
}

func ObserveKafkaConsumerHandleDuration(d time.Duration) {
	kafkaConsumerHandleDuration.Observe(d.Seconds())
}

func RecordKafkaConsumerHandlerRetry() {
	kafkaConsumerHandlerRetries.Inc()
}
