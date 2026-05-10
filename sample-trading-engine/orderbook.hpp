#pragma once

#include <map>
#include <vector>
#include <string>
#include <memory>
#include <chrono>
#include <shared_mutex>

/// Represents the side of an order.
enum class Side : char { Buy = 'B', Sell = 'S' };

/// Represents a single order in the orderbook.
struct Order {
    std::string id;
    std::string symbol;
    Side side;
    double price;
    int quantity;
    std::chrono::system_clock::time_point timestamp;
    
    Order(std::string id, std::string symbol, Side side, double price, int quantity)
        : id(std::move(id)), symbol(std::move(symbol)), side(side), price(price), quantity(quantity),
          timestamp(std::chrono::system_clock::now()) {}
};

/// Represents a completed trade between a buy and sell order.
struct Trade {
    std::string buy_order_id;
    std::string sell_order_id;
    std::string symbol;
    double price;
    int quantity;
    std::chrono::system_clock::time_point timestamp;
    
    Trade(std::string buy_id, std::string sell_id, std::string sym, double price, int qty)
        : buy_order_id(std::move(buy_id)), sell_order_id(std::move(sell_id)),
          symbol(std::move(sym)), price(price), quantity(qty),
          timestamp(std::chrono::system_clock::now()) {}
};

/// Price-time priority orderbook for a single symbol.
///
/// Thread-safe: all public methods acquire the appropriate lock.
/// Writers (addOrder, cancelOrder) take exclusive locks.
/// Readers (getBestBid, getBestAsk, getDepth, getTrades) take shared locks.
class OrderBook {
private:
    // Buy orders: price -> vector of orders (best bid at highest price)
    std::map<double, std::vector<std::shared_ptr<Order>>, std::greater<double>> buy_orders_;
    
    // Sell orders: price -> vector of orders (best ask at lowest price)
    std::map<double, std::vector<std::shared_ptr<Order>>> sell_orders_;
    
    std::vector<Trade> trades_;
    std::string symbol_;
    mutable std::shared_mutex mutex_;
    
public:
    explicit OrderBook(std::string symbol) : symbol_(std::move(symbol)) {}
    
    /// Add a new order to the orderbook. Returns any trades that resulted from matching.
    std::vector<Trade> addOrder(std::shared_ptr<Order> order);
    
    /// Cancel an existing order by ID. Returns true if the order was found and removed.
    bool cancelOrder(const std::string& order_id);
    
    /// Get aggregated buy-side depth (price -> total quantity).
    std::map<double, int> getBuyDepth() const;

    /// Get aggregated sell-side depth (price -> total quantity).
    std::map<double, int> getSellDepth() const;
    
    /// Get best bid price and quantity. Returns {0.0, 0} if no bids.
    std::pair<double, int> getBestBid() const;

    /// Get best ask price and quantity. Returns {0.0, 0} if no asks.
    std::pair<double, int> getBestAsk() const;
    
    /// Get a copy of all trades executed in this orderbook.
    std::vector<Trade> getTrades() const;
    
    /// Get the symbol this orderbook is for.
    const std::string& getSymbol() const { return symbol_; }
    
private:
    // Internal matching logic — caller must hold exclusive lock.
    std::vector<Trade> matchOrders(std::shared_ptr<Order> incoming_order);
};
