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

	// Lag from envelope event_time to successful handler completion (processing time vs event time).
	kafkaConsumerEventProcessingLag = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name: "kafka_consumer_event_processing_lag_seconds",
			Help: "Seconds between envelope event_time and successful processing (clamped to >= 0; skipped if event_time is zero).",
			Buckets: []float64{
				0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 300,
			},
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

// ObserveKafkaConsumerEventProcessingLag records time.Since(eventTime) at successful processing.
// Negative lag (clock skew) is clamped to zero. Zero eventTime is ignored.
func ObserveKafkaConsumerEventProcessingLag(eventTime time.Time) {
	if eventTime.IsZero() {
		return
	}
	lag := time.Since(eventTime)
	if lag < 0 {
		lag = 0
	}
	kafkaConsumerEventProcessingLag.Observe(lag.Seconds())
}
