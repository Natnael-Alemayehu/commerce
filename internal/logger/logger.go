package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-chi/chi/v5/middleware"
)

type contextKey struct{}

var loggerKey = contextKey{}

// New initializes a structured JSON slog.Logger.
func New(env string) *slog.Logger {
	var handler slog.Handler
	if env == "development" {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}
	return slog.New(handler)
}

// WithContext returns a new context with the given logger embedded.
func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext extracts the logger from the context, or returns a default logger.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// RequestLogger is a helper that creates a child logger with request ID for each HTTP request.
func RequestLogger(logger *slog.Logger, requestID string) *slog.Logger {
	return logger.With(slog.String("request_id", requestID))
}

// GetRequestID extracts the chi request ID from context.
func GetRequestID(ctx context.Context) string {
	return middleware.GetReqID(ctx)
}
