#pragma once

#include "orderbook.hpp"
#include <unordered_map>
#include <memory>
#include <string>
#include <vector>
#include <shared_mutex>

/// Central exchange managing multiple orderbooks.
///
/// Thread-safe: all public methods acquire the appropriate lock.
/// Each OrderBook also has its own internal mutex for fine-grained locking.
class Exchange {
private:
    mutable std::shared_mutex mutex_;
    std::unordered_map<std::string, std::unique_ptr<OrderBook>> orderbooks_;
    std::vector<Trade> all_trades_;
    
public:
    /// Add an order to the exchange. Creates the orderbook if it doesn't exist.
    std::vector<Trade> addOrder(std::shared_ptr<Order> order);
    
    /// Cancel an existing order by ID and symbol.
    bool cancelOrder(const std::string& order_id, const std::string& symbol);
    
    /// Get a raw pointer to the orderbook for a symbol. Returns nullptr if not found.
    OrderBook* getOrderBook(const std::string& symbol);
    
    /// Get a copy of all trades across all symbols.
    std::vector<Trade> getAllTrades() const;
    
    /// Create a new orderbook for the given symbol (no-op if it already exists).
    void createOrderBook(const std::string& symbol);
    
    /// List all active symbols.
    std::vector<std::string> getSymbols() const;
};
