package middleware

import (
	"net/http"
	"time"

	"starterkit/internal/logger"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

// RequestLogger logs incoming HTTP requests with structured fields.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		requestID := middleware.GetReqID(r.Context())
		log := logger.RequestLogger(logger.FromContext(r.Context()), requestID)
		ctx := logger.WithContext(r.Context(), log)

		next.ServeHTTP(ww, r.WithContext(ctx))

		log.Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"bytes", ww.BytesWritten(),
		)
	})
}

// RateLimit creates a per-IP rate limiter.
func RateLimit(requests int, window time.Duration) func(http.Handler) http.Handler {
	return httprate.LimitByIP(requests, window)
}
