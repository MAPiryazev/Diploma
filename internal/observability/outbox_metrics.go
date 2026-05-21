package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var outboxLatencyBuckets = []float64{
	0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1,
	0.25, 0.5, 1, 2.5, 5, 10,
}

var (
	outboxEventsPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "outbox_events_published_total",
			Help: "Outbox events successfully published and marked as sent.",
		},
		[]string{"event_type"},
	)

	outboxEventsFailed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "outbox_events_failed_total",
			Help: "Outbox events that failed publishing or marking.",
		},
		[]string{"event_type"},
	)

	outboxPublishDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "outbox_publish_duration_seconds",
			Help:    "Time spent publishing one outbox event and marking it as sent.",
			Buckets: outboxLatencyBuckets,
		},
		[]string{"event_type"},
	)
)

func RecordOutboxPublished(eventType string) {
	outboxEventsPublished.WithLabelValues(labelOrUnknown(eventType)).Inc()
}

func RecordOutboxPublishFailed(eventType string) {
	outboxEventsFailed.WithLabelValues(labelOrUnknown(eventType)).Inc()
}

func ObserveOutboxPublishDuration(eventType string, d time.Duration) {
	outboxPublishDuration.WithLabelValues(labelOrUnknown(eventType)).Observe(d.Seconds())
}
