package observability

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests handled by the service.",
		},
		[]string{"method", "route", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests handled by the service.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being processed.",
		},
	)

	transactionsCreatedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "transactions_created_total",
			Help: "Total number of successfully created transactions.",
		},
		[]string{"type", "status"},
	)

	analyticsRequestsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "analytics_requests_total",
			Help: "Total number of analytics requests accepted by the service.",
		},
	)
)

func IncHTTPRequestsInFlight() {
	httpRequestsInFlight.Inc()
}

func DecHTTPRequestsInFlight() {
	httpRequestsInFlight.Dec()
}

func ObserveHTTPRequest(method, route string, status int, duration time.Duration) {
	route = labelOrUnknown(route)

	httpRequestsTotal.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
	httpRequestDuration.WithLabelValues(method, route).Observe(duration.Seconds())
}

func RecordTransactionCreated(txType, status string) {
	transactionsCreatedTotal.WithLabelValues(labelOrUnknown(txType), labelOrUnknown(status)).Inc()
}

func RecordAnalyticsRequest() {
	analyticsRequestsTotal.Inc()
}

func labelOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}

	return value
}
