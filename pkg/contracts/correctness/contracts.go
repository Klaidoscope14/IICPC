package correctness

import "time"

// ExecutionReport represents a fill or order status update from the contestant engine.
// This is streamed via WebSocket (/ws/market-data).
type ExecutionReport struct {
	OrderID      string    `json:"order_id"`
	Symbol       string    `json:"symbol"`
	Side         string    `json:"side"` // "buy" or "sell"
	Status       string    `json:"status"` // "new", "partially_filled", "filled", "canceled", "rejected"
	FilledQty    int32     `json:"filled_qty"`
	LeavesQty    int32     `json:"leaves_qty"`
	Price        float64   `json:"price"` // The price at which the fill occurred
	Timestamp    time.Time `json:"timestamp"`
	MatchID      string    `json:"match_id,omitempty"` // Unique ID for the trade match
}

// TraceEventType defines the type of event in the trace log.
type TraceEventType string

const (
	TraceEventOrderSent      TraceEventType = "order_sent"
	TraceEventOrderAcked     TraceEventType = "order_acked"
	TraceEventExecution      TraceEventType = "execution_report"
)

// TraceEvent represents a single recorded action during the benchmark.
type TraceEvent struct {
	EventType TraceEventType   `json:"event_type"`
	Timestamp time.Time        `json:"timestamp"`
	
	// Present if EventType == "order_sent"
	OrderSent *OrderPayload    `json:"order_sent,omitempty"`
	
	// Present if EventType == "order_acked"
	OrderAcked *AckPayload     `json:"order_acked,omitempty"`
	
	// Present if EventType == "execution_report"
	Execution *ExecutionReport `json:"execution,omitempty"`
}

// OrderPayload represents the bot's outgoing REST request.
type OrderPayload struct {
	RequestID string  `json:"request_id"`
	OrderID   string  `json:"order_id"`
	Type      string  `json:"type"` // "limit", "market", "cancel"
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"`
	Price     float64 `json:"price,omitempty"`
	Quantity  int32   `json:"quantity,omitempty"`
}

// AckPayload represents the REST response received from the contestant engine.
type AckPayload struct {
	RequestID  string `json:"request_id"`
	StatusCode int    `json:"status_code"`
}
