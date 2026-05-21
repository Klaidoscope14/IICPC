package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/iicpc/websocket-service-go/internal/ws"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins since it's a dev setup. Production would restrict this.
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketHandler sets up the WebSocket routes and handles connections.
type WebSocketHandler struct {
	hub    *ws.Hub
	logger *slog.Logger
}

func NewWebSocketHandler(hub *ws.Hub, logger *slog.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		hub:    hub,
		logger: logger,
	}
}

// RegisterRoutes binds WebSocket endpoints to the router.
func (h *WebSocketHandler) RegisterRoutes(r *gin.Engine) {
	// Endpoint for live benchmark updates, telemetry, and container status.
	r.GET("/ws/benchmarks/:id/stream", h.streamBenchmark)
	
	// Global leaderboard stream (if requested without a specific benchmark)
	r.GET("/ws/leaderboard/stream", h.streamLeaderboard)
}

func (h *WebSocketHandler) streamBenchmark(c *gin.Context) {
	benchmarkID := c.Param("id")
	if benchmarkID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "benchmark_id is required"})
		return
	}
	h.serveWs(c, benchmarkID)
}

func (h *WebSocketHandler) streamLeaderboard(c *gin.Context) {
	// Empty string room represents the global room.
	h.serveWs(c, "")
}

func (h *WebSocketHandler) serveWs(c *gin.Context, room string) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to set websocket upgrade", slog.String("error", err.Error()))
		return
	}

	clientID := c.ClientIP() // Using IP for simplicity, in a real app this might be a session/user ID.
	client := ws.NewClient(clientID, room, h.hub, conn)

	// Register the client with the hub
	h.hub.Register(client)

	h.logger.Info("Client connected", slog.String("client_id", clientID), slog.String("room", room))

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.WritePump()
	go client.ReadPump()
}
