package bot

import (
	"fmt"
	"math/rand"
	"sync"
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

// Generator owns random state for one worker. rand.Rand is intentionally not
// shared across goroutines; sharing it would race and skew benchmark results.
type Generator struct {
	rng *rand.Rand
}

func NewGenerator(seed int64) *Generator {
	return &Generator{rng: rand.New(rand.NewSource(seed))}
}

var (
	defaultGeneratorMu sync.Mutex
	defaultGenerator   = NewGenerator(time.Now().UnixNano())
)

func (g *Generator) getSide(buyWeight int) Side {
	if g.rng.Intn(100) < buyWeight {
		return SideBuy
	}
	return SideSell
}

// LimitOrder generates a limit order.
func (g *Generator) LimitOrder(buyWeight int) Order {
	sym := symbols[g.rng.Intn(len(symbols))]
	side := g.getSide(buyWeight)
	basePrice := 100.0 + g.rng.Float64()*900.0
	price := float64(int(basePrice*100)) / 100 // round to 2dp
	qty := 1 + g.rng.Intn(100)

	return Order{
		Type:     OrderTypeLimit,
		Symbol:   sym,
		Side:     side,
		Price:    price,
		Quantity: qty,
	}
}

// MarketOrder generates a market order.
func (g *Generator) MarketOrder(buyWeight int) Order {
	sym := symbols[g.rng.Intn(len(symbols))]
	side := g.getSide(buyWeight)
	qty := 1 + g.rng.Intn(50)

	return Order{
		Type:     OrderTypeMarket,
		Symbol:   sym,
		Side:     side,
		Quantity: qty,
	}
}

// CancelOrder generates a cancel request for a fake order ID.
func (g *Generator) CancelOrder() Order {
	return Order{
		Type:    OrderTypeCancel,
		OrderID: fmt.Sprintf("ord-%d", g.rng.Int63()),
	}
}

// RandomOrder returns an order based on the provided profile weights.
func (g *Generator) RandomOrder(profile OrderProfile) Order {
	totalWeight := profile.LimitWeight + profile.MarketWeight + profile.CancelWeight
	if totalWeight <= 0 {
		// Fallback if bad profile
		return g.LimitOrder(50)
	}

	n := g.rng.Intn(totalWeight)
	switch {
	case n < profile.LimitWeight:
		return g.LimitOrder(profile.BuyWeight)
	case n < profile.LimitWeight+profile.MarketWeight:
		return g.MarketOrder(profile.BuyWeight)
	default:
		return g.CancelOrder()
	}
}

// LimitOrder generates a limit order using a package-level generator. New code
// should prefer a per-worker Generator to avoid lock contention.
func LimitOrder(buyWeight int) Order {
	defaultGeneratorMu.Lock()
	defer defaultGeneratorMu.Unlock()
	return defaultGenerator.LimitOrder(buyWeight)
}

// MarketOrder generates a market order using a package-level generator.
func MarketOrder(buyWeight int) Order {
	defaultGeneratorMu.Lock()
	defer defaultGeneratorMu.Unlock()
	return defaultGenerator.MarketOrder(buyWeight)
}

// CancelOrder generates a cancel request using a package-level generator.
func CancelOrder() Order {
	defaultGeneratorMu.Lock()
	defer defaultGeneratorMu.Unlock()
	return defaultGenerator.CancelOrder()
}

// RandomOrder returns an order using a package-level generator.
func RandomOrder(profile OrderProfile) Order {
	defaultGeneratorMu.Lock()
	defer defaultGeneratorMu.Unlock()
	return defaultGenerator.RandomOrder(profile)
}
