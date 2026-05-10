package logging

import (
	"context"
	"log/slog"
	"os"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// NewLogger creates a structured logger configured via LOG_LEVEL environment variable.
// Supported levels: debug, info, warn, error. Defaults to info.
func NewLogger(serviceName string) *slog.Logger {
	level := parseLevel(os.Getenv("LOG_LEVEL"))

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	})

	return slog.New(handler).With(slog.String("service", serviceName))
}

// NewDevLogger creates a human-readable text logger for local development.
func NewDevLogger(serviceName string) *slog.Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})

	return slog.New(handler).With(slog.String("service", serviceName))
}

// WithRequestID stores a request ID in the context for tracing.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext retrieves the request ID from the context, or empty string.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// LoggerFromContext returns a logger enriched with request-scoped fields.
func LoggerFromContext(ctx context.Context, base *slog.Logger) *slog.Logger {
	if id := RequestIDFromContext(ctx); id != "" {
		return base.With(slog.String("request_id", id))
	}
	return base
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
