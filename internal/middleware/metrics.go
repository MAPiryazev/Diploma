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

		observability.ObserveHTTPRequest(
			r.Method,
			normalizeRouteLabel(r.URL.Path),
			rw.statusCode,
			time.Since(start),
		)
	})
}

func normalizeRouteLabel(path string) string {
	switch {
	case path == "/":
		return "/"
	case path == "/health":
		return "/health"
	case path == "/metrics":
		return "/metrics"
	case path == "/analytics":
		return "/analytics"
	case path == "/items":
		return "/items"
	case strings.HasPrefix(path, "/items/"):
		return "/items/{id}"
	default:
		return "/static"
	}
}
