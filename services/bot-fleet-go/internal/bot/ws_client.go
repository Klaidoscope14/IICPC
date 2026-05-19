package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/iicpc/pkg/contracts/correctness"
)

// WSClient connects to the contestant's market-data websocket to receive execution reports.
type WSClient struct {
	serviceURL  string
	traceLogger *TraceLogger
	logger      *slog.Logger
	conn        *websocket.Conn
}

// NewWSClient creates a new WebSocket client.
func NewWSClient(serviceURL string, traceLogger *TraceLogger, logger *slog.Logger) *WSClient {
	return &WSClient{
		serviceURL:  serviceURL,
		traceLogger: traceLogger,
		logger:      logger,
	}
}

// Run connects to the WebSocket and streams messages until ctx is cancelled.
func (c *WSClient) Run(ctx context.Context) error {
	// Parse the service URL to build the ws:// or wss:// URL
	u, err := url.Parse(c.serviceURL)
	if err != nil {
		return fmt.Errorf("invalid service URL: %w", err)
	}

	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}

	wsURL := fmt.Sprintf("%s://%s/ws/market-data", scheme, u.Host)

	// Add retries for connecting, as the engine might just be starting up
	var conn *websocket.Conn
	var dialErr error
	for i := 0; i < 5; i++ {
		c.logger.Debug("dialing websocket", slog.String("url", wsURL), slog.Int("attempt", i+1))
		conn, _, dialErr = websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
		if dialErr == nil {
			break
		}
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}

	if dialErr != nil {
		c.logger.Warn("failed to connect to websocket, tracing will be incomplete", slog.String("error", dialErr.Error()))
		// We don't return the error, we just run without tracing fills if the engine doesn't support WS yet.
		return nil
	}

	c.conn = conn
	defer c.conn.Close()

	c.logger.Info("websocket connected for tracing", slog.String("url", wsURL))

	go func() {
		<-ctx.Done()
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Warn("websocket read error", slog.String("error", err.Error()))
			}
			return err
		}

		// Try to parse it as an ExecutionReport
		var execReport correctness.ExecutionReport
		if err := json.Unmarshal(message, &execReport); err != nil {
			// If it's a heartbeat or something else, ignore
			if !strings.Contains(string(message), "heartbeat") {
				c.logger.Debug("ignoring non-execution message", slog.String("msg", string(message)))
			}
			continue
		}

		// If it's an execution report, log it to the trace
		if c.traceLogger != nil && execReport.OrderID != "" {
			c.traceLogger.Log(correctness.TraceEvent{
				EventType: correctness.TraceEventExecution,
				Timestamp: time.Now(),
				Execution: &execReport,
			})
		}
	}
}
