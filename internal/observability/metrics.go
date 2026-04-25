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

	apiUseCaseRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_usecase_requests_total",
			Help: "Total number of API requests grouped by stable business use-case and result.",
		},
		[]string{"usecase", "result"},
	)

	apiUseCaseDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_usecase_duration_seconds",
			Help:    "Duration of API requests grouped by stable business use-case.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"usecase"},
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

func ObserveAPIUseCase(usecase, result string, duration time.Duration) {
	usecase = labelOrUnknown(usecase)
	result = labelOrUnknown(result)

	apiUseCaseRequests.WithLabelValues(usecase, result).Inc()
	apiUseCaseDuration.WithLabelValues(usecase).Observe(duration.Seconds())
}

func labelOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}

	return value
}
