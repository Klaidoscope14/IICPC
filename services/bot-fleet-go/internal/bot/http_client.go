package bot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

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
	serviceURL      string
	ordersURL       string
	cancelURLPrefix string
	httpClient      *http.Client
	traceLogger     *TraceLogger
}

// NewHTTPClient creates a bot HTTP client targeting the given service URL.
func NewHTTPClient(serviceURL string, timeoutMs int, traceLogger *TraceLogger) *HTTPClient {
	serviceURL = strings.TrimRight(serviceURL, "/")
	return &HTTPClient{
		serviceURL:      serviceURL,
		ordersURL:       serviceURL + "/api/orders",
		cancelURLPrefix: serviceURL + "/api/orders/",
		httpClient: &http.Client{
			Timeout:   time.Duration(timeoutMs) * time.Millisecond,
			Transport: sharedTransport,
		},
		traceLogger: traceLogger,
	}
}

// Send dispatches a single order and returns the tracking result.
func (c *HTTPClient) Send(ctx context.Context, order Order) Result {
	requestID := nextRequestID()
	sentAt := time.Now()

	result := Result{
		RequestID: requestID,
		OrderType: order.Type,
		SentAt:    sentAt,
	}

	body := encodeOrder(order)

	var (
		method string
		url    string
	)
	if order.Type == OrderTypeCancel {
		method = http.MethodDelete
		url = c.cancelURLPrefix + order.OrderID
	} else {
		method = http.MethodPost
		url = c.ordersURL
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

var (
	requestSeq    atomic.Uint64
	requestPrefix = strconv.FormatInt(time.Now().UnixNano(), 36)
)

var sharedTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   2 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	MaxIdleConns:          4096,
	MaxIdleConnsPerHost:   1024,
	MaxConnsPerHost:       1024,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   2 * time.Second,
	ExpectContinueTimeout: 250 * time.Millisecond,
	ResponseHeaderTimeout: 0,
	DisableCompression:    true,
}

func nextRequestID() string {
	return requestPrefix + "-" + strconv.FormatUint(requestSeq.Add(1), 36)
}

func encodeOrder(order Order) []byte {
	body := make([]byte, 0, 128)
	body = append(body, '{')
	body = appendStringField(body, "type", string(order.Type), false)
	body = appendStringField(body, "symbol", string(order.Symbol), true)
	if order.Side != "" {
		body = appendStringField(body, "side", string(order.Side), true)
	}
	if order.Price != 0 {
		body = appendNumberField(body, "price", strconv.AppendFloat(nil, order.Price, 'f', -1, 64), true)
	}
	if order.Quantity != 0 {
		body = appendNumberField(body, "quantity", strconv.AppendInt(nil, int64(order.Quantity), 10), true)
	}
	if order.OrderID != "" {
		body = appendStringField(body, "order_id", order.OrderID, true)
	}
	body = append(body, '}')
	return body
}

func appendStringField(dst []byte, name string, value string, comma bool) []byte {
	if comma {
		dst = append(dst, ',')
	}
	dst = strconv.AppendQuote(dst, name)
	dst = append(dst, ':')
	dst = strconv.AppendQuote(dst, value)
	return dst
}

func appendNumberField(dst []byte, name string, value []byte, comma bool) []byte {
	if comma {
		dst = append(dst, ',')
	}
	dst = strconv.AppendQuote(dst, name)
	dst = append(dst, ':')
	dst = append(dst, value...)
	return dst
}
