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
				log.Printf("[PANIC] %v on %s %s", err, r.Method, r.RequestURI)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
