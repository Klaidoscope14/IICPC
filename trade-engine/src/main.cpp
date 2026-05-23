#include "Exchange.h"
#include <iostream>
#include <thread>
#include <chrono>

using namespace Mercury;

// Helper to construct a limit order
static Order makeLimitOrder(uint64_t id, Side side, int64_t price, uint64_t quantity, uint64_t clientId) {
    Order o;
    o.id = id;
    o.orderType = OrderType::Limit;
    o.side = side;
    o.price = price;
    o.quantity = quantity;
    o.clientId = clientId;
    return o;
}

void printMarketDepth(Exchange& exchange, const std::string& symbol) {
    auto snapshot = exchange.getSnapshot(symbol, 5);
    std::cout << "\n=== " << symbol << " Market Depth ===" << std::endl;
    
    // Print asks (reverse order so lowest is closest to mid)
    for (auto it = snapshot.asks.rbegin(); it != snapshot.asks.rend(); ++it) {
        std::cout << "ASK: " << it->price << " x" << it->quantity << " (" << it->orderCount << " orders)\n";
    }
    
    std::cout << "------------------------\n";
    
    // Print bids
    for (const auto& level : snapshot.bids) {
        std::cout << "BID: " << level.price << " x" << level.quantity << " (" << level.orderCount << " orders)\n";
    }
}

int main() {
    std::cout << "Starting Simple IICPC Trading Engine...\n";
    
    Exchange exchange;
    
    // Initialize order books
    exchange.createOrderBook("AAPL");
    exchange.createOrderBook("GOOGL");
    
    // Submit some orders
    uint64_t orderId = 1;
    
    // Add resting liquidity
    exchange.submitOrder("AAPL", makeLimitOrder(orderId++, Side::Buy, 15000, 100, 1));
    exchange.submitOrder("AAPL", makeLimitOrder(orderId++, Side::Buy, 14900, 50, 1));
    exchange.submitOrder("AAPL", makeLimitOrder(orderId++, Side::Sell, 15100, 100, 2));
    exchange.submitOrder("AAPL", makeLimitOrder(orderId++, Side::Sell, 15200, 200, 2));
    
    printMarketDepth(exchange, "AAPL");
    
    // Aggressive order matching
    std::cout << "\nSubmitting aggressive buy order for AAPL at 15100 x 50...\n";
    auto result = exchange.submitOrder("AAPL", makeLimitOrder(orderId++, Side::Buy, 15100, 50, 3));
    
    if (result.status == ExecutionStatus::Filled || result.status == ExecutionStatus::Resting) {
        std::cout << "Order processed. Fills: " << result.trades.size() << "\n";
        for (const auto& trade : result.trades) {
            std::cout << "  Trade: Price " << trade.price << ", Qty " << trade.quantity << "\n";
        }
    }
    
    printMarketDepth(exchange, "AAPL");
    
    std::cout << "\nEngine run complete.\n";
    return 0;
}
