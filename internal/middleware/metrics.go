package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/observability"
)

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		observability.IncHTTPRequestsInFlight()
		defer observability.DecHTTPRequestsInFlight()

		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		route := normalizeRouteLabel(r.URL.Path)
		observability.ObserveHTTPRequest(
			r.Method,
			route,
			rw.statusCode,
			duration,
		)
		if usecase := useCaseLabel(r.Method, route); usecase != "" {
			observability.ObserveAPIUseCase(usecase, resultLabel(rw.statusCode), duration)
		}
	})
}

func normalizeRouteLabel(path string) string {
	switch {
	case path == "/":
		return "/"
	case path == "/health":
		return "/health"
	case path == "/ready":
		return "/ready"
	case path == "/metrics":
		return "/metrics"
	case path == "/analytics":
		return "/analytics"
	case path == "/analytics/stream":
		return "/analytics/stream"
	case path == "/monitoring/alerts":
		return "/monitoring/alerts"
	case path == "/items":
		return "/items"
	case strings.HasPrefix(path, "/items/"):
		return "/items/{id}"
	default:
		return "/static"
	}
}

func useCaseLabel(method, route string) string {
	switch method + " " + route {
	case "POST /items":
		return "transaction.create"
	case "GET /items":
		return "transaction.list"
	case "GET /items/{id}":
		return "transaction.get"
	case "PUT /items/{id}":
		return "transaction.update"
	case "DELETE /items/{id}":
		return "transaction.delete"
	case "GET /analytics":
		return "analytics.get"
	case "GET /analytics/stream":
		return "analytics.stream"
	case "GET /monitoring/alerts":
		return "monitoring.alerts"
	default:
		return ""
	}
}

func resultLabel(status int) string {
	if status >= 200 && status < 400 {
		return "success"
	}
	return "error"
}
