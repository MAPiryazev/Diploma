package middleware

import (
	"fmt"
	"log"
	"net/http"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, `{"status":"error","message":"internal server error"}`)
				log.Printf(
					"level=error msg=%q method=%s path=%q request_id=%s panic=%v",
					"recovered from panic",
					r.Method,
					r.RequestURI,
					GetRequestID(r.Context()),
					err,
				)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
