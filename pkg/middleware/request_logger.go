package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iicpc/pkg/logging"
	"log/slog"
)

// RequestLogger returns a Gin middleware that logs every request with structured fields.
// Attaches a unique request ID to the context for tracing.
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Generate and attach request ID.
		requestID := uuid.New().String()
		ctx := logging.WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Header("X-Request-ID", requestID)

		// Process request.
		c.Next()

		// Log after response.
		duration := time.Since(start)
		logger.Info("request",
			slog.String("request_id", requestID),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("duration", duration),
			slog.String("client_ip", c.ClientIP()),
		)
	}
}
