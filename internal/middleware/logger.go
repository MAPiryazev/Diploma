package middleware

import (
	"log"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(data)
}

type logEntry struct {
	Method     string
	Path       string
	StatusCode int
	DurationMs int64
	Timestamp  string
	ClientIP   string
	UserAgent  string
	RequestID  string
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		requestID := GetRequestID(r.Context())
		entry := logEntry{
			Method:     r.Method,
			Path:       r.RequestURI,
			StatusCode: rw.statusCode,
			DurationMs: duration.Milliseconds(),
			Timestamp:  time.Now().Format(time.RFC3339),
			ClientIP:   r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			RequestID:  requestID,
		}

		log.Printf(
			"level=info method=%s path=%q status=%d duration_ms=%d client_ip=%q user_agent=%q request_id=%s",
			entry.Method,
			entry.Path,
			entry.StatusCode,
			entry.DurationMs,
			entry.ClientIP,
			entry.UserAgent,
			entry.RequestID,
		)
	})
}
