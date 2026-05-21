package bot

import (
	"fmt"
	"math/rand"
	"time"
)

// OrderProfile dictates the probability distribution of order types and sides.
type OrderProfile struct {
	LimitWeight  int
	MarketWeight int
	CancelWeight int
	BuyWeight    int // Out of 100
}

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

func getSide(buyWeight int) Side {
	if rng.Intn(100) < buyWeight {
		return SideBuy
	}
	return SideSell
}

// LimitOrder generates a limit order.
func LimitOrder(buyWeight int) Order {
	sym := symbols[rng.Intn(len(symbols))]
	side := getSide(buyWeight)
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
func MarketOrder(buyWeight int) Order {
	sym := symbols[rng.Intn(len(symbols))]
	side := getSide(buyWeight)
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

// RandomOrder returns an order based on the provided profile weights.
func RandomOrder(profile OrderProfile) Order {
	totalWeight := profile.LimitWeight + profile.MarketWeight + profile.CancelWeight
	if totalWeight <= 0 {
		// Fallback if bad profile
		return LimitOrder(50)
	}

	n := rng.Intn(totalWeight)
	switch {
	case n < profile.LimitWeight:
		return LimitOrder(profile.BuyWeight)
	case n < profile.LimitWeight+profile.MarketWeight:
		return MarketOrder(profile.BuyWeight)
	default:
		return CancelOrder()
	}
}
