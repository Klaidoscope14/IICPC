package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/iicpc/benchmark-orchestrator-go/internal/service"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in dev. Restrict in production.
	},
}

// WebSocketHandler handles WebSocket connections for live benchmark telemetry.
type WebSocketHandler struct {
	service service.OrchestratorService
	logger  *slog.Logger
}

// NewWebSocketHandler creates a WebSocket handler wired to the orchestrator service.
func NewWebSocketHandler(svc service.OrchestratorService, logger *slog.Logger) *WebSocketHandler {
	return &WebSocketHandler{service: svc, logger: logger}
}

// RegisterRoutes binds WebSocket endpoints to the router.
func (h *WebSocketHandler) RegisterRoutes(r *gin.Engine) {
	r.GET("/ws/benchmarks/:id/stream", h.StreamBenchmark)
}

// StreamBenchmark upgrades the connection to WebSocket and streams benchmark metrics.
func (h *WebSocketHandler) StreamBenchmark(c *gin.Context) {
	benchmarkID := c.Param("id")
	if benchmarkID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "benchmark_id is required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed",
			slog.String("error", err.Error()),
			slog.String("benchmark_id", benchmarkID),
		)
		return
	}
	defer conn.Close()

	h.logger.Info("websocket client connected",
		slog.String("benchmark_id", benchmarkID),
		slog.String("client", c.ClientIP()),
	)

	// Stream metrics at 1-second intervals.
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Listen for client close in a separate goroutine.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-done:
			h.logger.Info("websocket client disconnected",
				slog.String("benchmark_id", benchmarkID),
			)
			return
		case <-ticker.C:
			benchmark, err := h.service.GetBenchmarkStatus(c.Request.Context(), benchmarkID)
			if err != nil {
				conn.WriteJSON(gin.H{"error": err.Error()})
				return
			}

			// Send the benchmark status (includes metrics).
			if err := conn.WriteJSON(benchmark); err != nil {
				h.logger.Debug("websocket write failed",
					slog.String("error", err.Error()),
				)
				return
			}

			// Stop streaming if the benchmark is completed or failed.
			if benchmark.Status == "completed" || benchmark.Status == "failed" || benchmark.Status == "stopped" {
				conn.WriteJSON(gin.H{"event": "benchmark_ended", "status": string(benchmark.Status)})
				return
			}
		}
	}
}
