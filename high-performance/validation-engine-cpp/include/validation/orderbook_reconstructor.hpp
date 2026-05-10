#pragma once

#include "validation/types.hpp"
#include <map>
#include <vector>
#include <memory>
#include <unordered_map>

namespace validation {

/// Reconstructs an orderbook from a sequence of LogOrders.
///
/// Maintains the expected state of the orderbook at any point in time,
/// which validators use as ground truth.
class OrderbookReconstructor {
public:
    explicit OrderbookReconstructor(const std::string& symbol);

    /// Process an incoming order and update the book state.
    void apply_order(const LogOrder& order);

    /// Process a cancel and remove the order from the book.
    void apply_cancel(const std::string& order_id);

    /// Remove filled quantity from a resting order.
    void apply_fill(const std::string& order_id, int32_t fill_qty);

    /// Get the current best bid price, or 0 if empty.
    double best_bid() const;

    /// Get the current best ask price, or 0 if empty.
    double best_ask() const;

    /// Get the order at the front of the queue at a given price level (price-time priority).
    const LogOrder* front_at_price(Side side, double price) const;

    /// Check if an order exists in the book.
    bool has_order(const std::string& order_id) const;

    /// Get a specific order by ID.
    const LogOrder* get_order(const std::string& order_id) const;

    /// Get the symbol this book is for.
    const std::string& symbol() const { return symbol_; }

private:
    std::string symbol_;

    // Buy side: price -> orders (highest price first, FIFO within price).
    std::map<double, std::vector<LogOrder>, std::greater<double>> bids_;

    // Sell side: price -> orders (lowest price first, FIFO within price).
    std::map<double, std::vector<LogOrder>> asks_;

    // Order lookup by ID.
    std::unordered_map<std::string, std::pair<Side, double>> order_index_;
};

} // namespace validation
