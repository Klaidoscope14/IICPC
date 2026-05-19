package correctness

import (
	"sort"
	"strings"
	"time"

	"github.com/iicpc/pkg/contracts/correctness"
)

// Order represents an order in the shadow orderbook.
type Order struct {
	ID        string
	Symbol    string
	Side      string // "buy" or "sell"
	Type      string // "limit" or "market"
	Price     float64
	Qty       int32
	LeavesQty int32
	Timestamp time.Time
}

// Orderbook maintains the state of orders and simulates matching.
type Orderbook struct {
	Bids []*Order // Sorted descending by price, ascending by time
	Asks []*Order // Sorted ascending by price, ascending by time
	
	ordersByID map[string]*Order
}

func NewOrderbook() *Orderbook {
	return &Orderbook{
		Bids:       make([]*Order, 0),
		Asks:       make([]*Order, 0),
		ordersByID: make(map[string]*Order),
	}
}

// ExpectedMatch represents a trade that *should* happen.
type ExpectedMatch struct {
	MakerOrderID string
	TakerOrderID string
	Price        float64
	Qty          int32
}

// ProcessOrder adds an order to the book and returns expected matches.
func (ob *Orderbook) ProcessOrder(payload *correctness.OrderPayload, timestamp time.Time) []ExpectedMatch {
	payloadType := strings.ToLower(payload.Type)
	if payloadType == "cancel" {
		ob.CancelOrder(payload.OrderID)
		return nil
	}

	order := &Order{
		ID:        payload.OrderID,
		Symbol:    payload.Symbol,
		Side:      strings.ToLower(payload.Side),
		Type:      payloadType,
		Price:     payload.Price,
		Qty:       payload.Quantity,
		LeavesQty: payload.Quantity,
		Timestamp: timestamp,
	}

	ob.ordersByID[order.ID] = order

	var matches []ExpectedMatch

	// Match the order
	if order.Side == "buy" {
		matches = ob.matchBuy(order)
		if order.LeavesQty > 0 && order.Type == "limit" {
			ob.insertBid(order)
		}
	} else if order.Side == "sell" {
		matches = ob.matchSell(order)
		if order.LeavesQty > 0 && order.Type == "limit" {
			ob.insertAsk(order)
		}
	}

	return matches
}

func (ob *Orderbook) CancelOrder(orderID string) {
	if order, exists := ob.ordersByID[orderID]; exists {
		order.LeavesQty = 0 // Logically cancel it
		// For simplicity, we just leave it in the slice with 0 qty.
		// It will be cleaned up during matching.
	}
}

func (ob *Orderbook) matchBuy(buyOrder *Order) []ExpectedMatch {
	var matches []ExpectedMatch
	var newAsks []*Order

	for _, ask := range ob.Asks {
		if buyOrder.LeavesQty == 0 {
			newAsks = append(newAsks, ask)
			continue
		}
		if ask.LeavesQty == 0 {
			continue // skip cancelled or fully filled
		}

		// Price check
		if buyOrder.Type == "limit" && buyOrder.Price < ask.Price {
			newAsks = append(newAsks, ask)
			continue
		}

		// Match
		matchQty := buyOrder.LeavesQty
		if ask.LeavesQty < matchQty {
			matchQty = ask.LeavesQty
		}

		matches = append(matches, ExpectedMatch{
			MakerOrderID: ask.ID,
			TakerOrderID: buyOrder.ID,
			Price:        ask.Price, // Maker price
			Qty:          matchQty,
		})

		buyOrder.LeavesQty -= matchQty
		ask.LeavesQty -= matchQty

		if ask.LeavesQty > 0 {
			newAsks = append(newAsks, ask)
		}
	}
	
	// Keep the rest of the asks
	ob.Asks = newAsks
	return matches
}

func (ob *Orderbook) matchSell(sellOrder *Order) []ExpectedMatch {
	var matches []ExpectedMatch
	var newBids []*Order

	for _, bid := range ob.Bids {
		if sellOrder.LeavesQty == 0 {
			newBids = append(newBids, bid)
			continue
		}
		if bid.LeavesQty == 0 {
			continue
		}

		// Price check
		if sellOrder.Type == "limit" && sellOrder.Price > bid.Price {
			newBids = append(newBids, bid)
			continue
		}

		// Match
		matchQty := sellOrder.LeavesQty
		if bid.LeavesQty < matchQty {
			matchQty = bid.LeavesQty
		}

		matches = append(matches, ExpectedMatch{
			MakerOrderID: bid.ID,
			TakerOrderID: sellOrder.ID,
			Price:        bid.Price, // Maker price
			Qty:          matchQty,
		})

		sellOrder.LeavesQty -= matchQty
		bid.LeavesQty -= matchQty

		if bid.LeavesQty > 0 {
			newBids = append(newBids, bid)
		}
	}

	ob.Bids = newBids
	return matches
}

func (ob *Orderbook) insertBid(order *Order) {
	// Insert maintaining descending price, ascending time
	idx := sort.Search(len(ob.Bids), func(i int) bool {
		// return true if element at i should be AFTER our order
		if ob.Bids[i].Price != order.Price {
			return ob.Bids[i].Price < order.Price
		}
		return ob.Bids[i].Timestamp.After(order.Timestamp)
	})

	ob.Bids = append(ob.Bids, nil)
	copy(ob.Bids[idx+1:], ob.Bids[idx:])
	ob.Bids[idx] = order
}

func (ob *Orderbook) insertAsk(order *Order) {
	// Insert maintaining ascending price, ascending time
	idx := sort.Search(len(ob.Asks), func(i int) bool {
		// return true if element at i should be AFTER our order
		if ob.Asks[i].Price != order.Price {
			return ob.Asks[i].Price > order.Price
		}
		return ob.Asks[i].Timestamp.After(order.Timestamp)
	})

	ob.Asks = append(ob.Asks, nil)
	copy(ob.Asks[idx+1:], ob.Asks[idx:])
	ob.Asks[idx] = order
}
