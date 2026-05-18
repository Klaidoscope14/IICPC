package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// GetLogFilePath resolves the absolute path to the central log file.
func GetLogFilePath() string {
	if envPath := os.Getenv("LOG_FILE_PATH"); envPath != "" {
		return envPath
	}

	dir, err := os.Getwd()
	if err != nil {
		return "logs/iicpc.log"
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return filepath.Join(dir, "logs", "iicpc.log")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "logs/iicpc.log"
}

// NewLogger creates a structured logger configured via LOG_LEVEL environment variable.
// Supported levels: debug, info, warn, error. Defaults to info.
func NewLogger(serviceName string) *slog.Logger {
	level := parseLevel(os.Getenv("LOG_LEVEL"))

	// Ensure logs directory exists (relative to workspace root for local dev MVP)
	logPath := GetLogFilePath()
	os.MkdirAll(filepath.Dir(logPath), 0755)
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	var writer io.Writer
	if err != nil {
		writer = os.Stdout
	} else {
		writer = io.MultiWriter(os.Stdout, file)
	}

	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
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

// WithBenchmarkID stores a benchmark ID in the context for tracing.
func WithBenchmarkID(ctx context.Context, benchmarkID string) context.Context {
	return context.WithValue(ctx, contextKey("benchmark_id"), benchmarkID)
}

// WithSubmissionID stores a submission ID in the context for tracing.
func WithSubmissionID(ctx context.Context, submissionID string) context.Context {
	return context.WithValue(ctx, contextKey("submission_id"), submissionID)
}

// LoggerFromContext returns a logger enriched with request-scoped fields.
func LoggerFromContext(ctx context.Context, base *slog.Logger) *slog.Logger {
	l := base
	if id := RequestIDFromContext(ctx); id != "" {
		l = l.With(slog.String("request_id", id))
	}
	if id, ok := ctx.Value(contextKey("benchmark_id")).(string); ok {
		l = l.With(slog.String("benchmark_id", id))
	}
	if id, ok := ctx.Value(contextKey("submission_id")).(string); ok {
		l = l.With(slog.String("submission_id", id))
	}
	return l
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
