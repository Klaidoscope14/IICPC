package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/iicpc/pkg/contracts/correctness"
)

// Result captures the outcome of a single bot request.
type Result struct {
	RequestID  string
	OrderType  OrderType
	StatusCode int
	LatencyMs  float64
	SentAt     time.Time
	AckAt      time.Time
	Err        error
	TimedOut   bool
}

// HTTPClient is a bot that fires HTTP requests at the contestant's engine.
type HTTPClient struct {
	serviceURL  string
	httpClient  *http.Client
	traceLogger *TraceLogger
}

// NewHTTPClient creates a bot HTTP client targeting the given service URL.
func NewHTTPClient(serviceURL string, timeoutMs int, traceLogger *TraceLogger) *HTTPClient {
	return &HTTPClient{
		serviceURL: serviceURL,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutMs) * time.Millisecond,
		},
		traceLogger: traceLogger,
	}
}

// Send dispatches a single order and returns the tracking result.
func (c *HTTPClient) Send(ctx context.Context, order Order) Result {
	requestID := uuid.New().String()
	sentAt := time.Now()

	result := Result{
		RequestID: requestID,
		OrderType: order.Type,
		SentAt:    sentAt,
	}

	body, err := json.Marshal(order)
	if err != nil {
		result.Err = fmt.Errorf("marshal error: %w", err)
		return result
	}

	var (
		method string
		url    string
	)
	if order.Type == OrderTypeCancel {
		method = http.MethodDelete
		url = fmt.Sprintf("%s/api/orders/%s", c.serviceURL, order.OrderID)
	} else {
		method = http.MethodPost
		url = fmt.Sprintf("%s/api/orders", c.serviceURL)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		result.Err = fmt.Errorf("request build error: %w", err)
		return result
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", requestID)

	if c.traceLogger != nil {
		c.traceLogger.Log(correctness.TraceEvent{
			EventType: correctness.TraceEventOrderSent,
			Timestamp: sentAt,
			OrderSent: &correctness.OrderPayload{
				RequestID: requestID,
				OrderID:   order.OrderID,
				Type:      string(order.Type),
				Symbol:    string(order.Symbol),
				Side:      string(order.Side),
				Price:     order.Price,
				Quantity:  int32(order.Quantity),
			},
		})
	}

	resp, err := c.httpClient.Do(req)
	ackAt := time.Now()
	result.AckAt = ackAt
	result.LatencyMs = float64(ackAt.Sub(sentAt).Microseconds()) / 1000.0

	if err != nil {
		if ctx.Err() != nil {
			result.TimedOut = true
			result.Err = fmt.Errorf("timeout: %w", ctx.Err())
		} else {
			result.Err = fmt.Errorf("http error: %w", err)
		}
		return result
	}
	defer func() { io.Copy(io.Discard, resp.Body); resp.Body.Close() }()

	result.StatusCode = resp.StatusCode
	
	if c.traceLogger != nil {
		c.traceLogger.Log(correctness.TraceEvent{
			EventType: correctness.TraceEventOrderAcked,
			Timestamp: ackAt,
			OrderAcked: &correctness.AckPayload{
				RequestID:  requestID,
				StatusCode: result.StatusCode,
			},
		})
	}
	
	return result
}
