package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		log.Printf(
			"method=%s path=%s requestId=%s duration=%s",
			r.Method,
			r.URL.Path,
			r.Header.Get("X-Request-ID"),
			time.Since(start),
		)
	})
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()
		r.Header.Set("X-Request-ID", requestID)
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r)
	})
}
