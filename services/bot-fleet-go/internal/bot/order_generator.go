package bot

import (
	"fmt"
	"math/rand"
	"time"
)

// Symbol is a trading pair traded by bots.
type Symbol string

var symbols = []Symbol{"AAPL", "GOOG", "MSFT", "TSLA", "AMZN"}

// Side is BUY or SELL.
type Side string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

// OrderType categorises the generated order.
type OrderType string

const (
	OrderTypeLimit  OrderType = "LIMIT"
	OrderTypeMarket OrderType = "MARKET"
	OrderTypeCancel OrderType = "CANCEL"
)

// Order represents a generated trading order payload.
type Order struct {
	Type     OrderType `json:"type"`
	Symbol   Symbol    `json:"symbol"`
	Side     Side      `json:"side,omitempty"`
	Price    float64   `json:"price,omitempty"`
	Quantity int       `json:"quantity,omitempty"`
	OrderID  string    `json:"order_id,omitempty"` // for cancel
}

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// LimitOrder generates a limit order.
func LimitOrder() Order {
	sym := symbols[rng.Intn(len(symbols))]
	side := SideBuy
	if rng.Intn(2) == 1 {
		side = SideSell
	}
	basePrice := 100.0 + rng.Float64()*900.0
	price := float64(int(basePrice*100)) / 100 // round to 2dp
	qty := 1 + rng.Intn(100)

	return Order{
		Type:     OrderTypeLimit,
		Symbol:   sym,
		Side:     side,
		Price:    price,
		Quantity: qty,
	}
}

// MarketOrder generates a market order.
func MarketOrder() Order {
	sym := symbols[rng.Intn(len(symbols))]
	side := SideBuy
	if rng.Intn(2) == 1 {
		side = SideSell
	}
	qty := 1 + rng.Intn(50)

	return Order{
		Type:     OrderTypeMarket,
		Symbol:   sym,
		Side:     side,
		Quantity: qty,
	}
}

// CancelOrder generates a cancel request for a fake order ID.
func CancelOrder() Order {
	return Order{
		Type:    OrderTypeCancel,
		OrderID: fmt.Sprintf("ord-%d", rng.Int63()),
	}
}

// RandomOrder returns a random order type with realistic weights:
// 50% limit, 30% market, 20% cancel.
func RandomOrder() Order {
	n := rng.Intn(100)
	switch {
	case n < 50:
		return LimitOrder()
	case n < 80:
		return MarketOrder()
	default:
		return CancelOrder()
	}
}
